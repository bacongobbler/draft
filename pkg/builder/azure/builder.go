package azure

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/draft/pkg/azure/blob"
	"github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2019-12-01-preview/containerregistry"
	"github.com/Azure/draft/pkg/builder"
	"github.com/Azure/go-autorest/autorest/adal"
	azurecli "github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

// Builder contains information about the build environment
type Builder struct {
	RegistryClient containerregistry.RegistriesClient
	RunsClient   containerregistry.RunsClient
	AdalToken      adal.Token
	Subscription   azurecli.Subscription
}

// Build builds the docker image.
func (b *Builder) Build(ctx context.Context, app *builder.AppContext, out chan<- *builder.Summary) (err error) {
	const stageDesc = "Building Docker Image"

	defer builder.Complete(app.ID, stageDesc, out, &err)
	summary := builder.Summarize(app.ID, stageDesc, out)

	// notify that particular stage has started.
	summary("started", builder.SummaryStarted)

	msgc := make(chan string)
	errc := make(chan error)
	go func() {
		defer func() {
			close(msgc)
			close(errc)
		}()
		// the azure SDK wants only the name of the registry rather than the full registry URL
		registryName := getRegistryName(app.Ctx.Env.Registry)
		// first, upload the tarball to the upload storage URL given to us by acr build
		sourceUploadDefinition, err := b.RegistryClient.GetBuildSourceUploadURL(ctx, app.Ctx.Env.ResourceGroupName, registryName)
		if err != nil {
			errc <- fmt.Errorf("Could not retrieve acr build's upload URL: %v", err)
			return
		}
		u, err := url.Parse(*sourceUploadDefinition.UploadURL)
		if err != nil {
			errc <- fmt.Errorf("Could not parse blob upload URL: %v", err)
			return
		}

		blockBlobService := azblob.NewBlockBlobURL(*u, azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{}))
		// Upload the application tarball to acr build
		_, err = blockBlobService.Upload(ctx, bytes.NewReader(app.Ctx.Archive), azblob.BlobHTTPHeaders{ContentType: "application/gzip"}, azblob.Metadata{}, azblob.BlobAccessConditions{})
		if err != nil {
			errc <- fmt.Errorf("Could not upload docker context to acr build: %v", err)
			return
		}

		var imageNames []string
		for i := range app.Images {
			imageNameParts := strings.Split(app.Images[i], ":")
			// get the tag name from the image name
			imageNames = append(imageNames, fmt.Sprintf("%s:%s", app.Ctx.Env.Name, imageNameParts[len(imageNameParts)-1]))
		}

		var args []containerregistry.Argument

		for k := range app.Ctx.Env.ImageBuildArgs {
			name := k
			value := app.Ctx.Env.ImageBuildArgs[k]
			arg := containerregistry.Argument{
				Name:  &name,
				Value: &value,
				IsSecret: to.BoolPtr(false),
			}
			args = append(args, arg)
		}

		req := containerregistry.DockerBuildRequest{
			ImageNames:     to.StringSlicePtr(imageNames),
			SourceLocation: sourceUploadDefinition.RelativePath,
			Arguments: &args,
			IsPushEnabled:  to.BoolPtr(true),
			Timeout:        to.Int32Ptr(600),
			Platform: &containerregistry.PlatformProperties{
				// TODO: make this configurable once ACR build supports windows containers
				Os: containerregistry.Linux,
				Architecture: containerregistry.Amd64,
				// NB: CPU isn't required right now, possibly want to make this configurable
				// It'll actually default to 2 from the server
				// CPU: to.Int32Ptr(1),
			},
			// TODO: make this configurable
			DockerFilePath: to.StringPtr("Dockerfile"),
			Type:           containerregistry.TypeDockerBuildRequest,
		}
		bas, ok := req.AsBasicRunRequest()
		if !ok {
			errc <- errors.New("Failed to create quick build request")
			return
		}
		future, err := b.RegistryClient.ScheduleRun(ctx, app.Ctx.Env.ResourceGroupName, registryName, bas)
		if err != nil {
			errc <- fmt.Errorf("Could not while queue acr build: %v", err)
			return
		}

		if err := future.WaitForCompletionRef(ctx, b.RegistryClient.Client); err != nil {
			errc <- fmt.Errorf("Could not wait for acr build to complete: %v", err)
			return
		}

		fin, err := future.Result(b.RegistryClient)
		if err != nil {
			errc <- fmt.Errorf("Could not retrieve acr build future result: %v", err)
			return
		}

		logResult, err := b.RunsClient.GetLogSasURL(ctx, app.Ctx.Env.ResourceGroupName, registryName, *fin.ID)
		if err != nil {
			errc <- fmt.Errorf("Could not retrieve build log SAS URL: %v", err)
			return
		}

		if *logResult.LogLink == "" {
			errc <- errors.New("Unable to create a link to the logs: no link found")
			return
		}

		blobURL := blob.GetAppendBlobURL(*logResult.LogLink)

		get, err := blobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)
		if err != nil {
			errc <- fmt.Errorf("Could not retrieve build logs: %v", err)
			return
		}

		reader := get.Body(azblob.RetryReaderOptions{})
		defer reader.Close()

		_, err = io.Copy(app.Log, reader)
		if err != nil {
			errc <- fmt.Errorf("Could not stream build logs: %v", err)
			return
		}

		return

	}()
	for msgc != nil || errc != nil {
		select {
		case msg, ok := <-msgc:
			if !ok {
				msgc = nil
				continue
			}
			summary(msg, builder.SummaryLogging)
		case err, ok := <-errc:
			if !ok {
				errc = nil
				continue
			}
			return err
		default:
			summary("ongoing", builder.SummaryOngoing)
			time.Sleep(time.Second)
		}
	}
	return nil
}

// Push pushes the results of Build to the image repository.
func (b *Builder) Push(ctx context.Context, app *builder.AppContext, out chan<- *builder.Summary) (err error) {
	// no-op: acr build pushes to the registry through the quickbuild request
	const stageDesc = "Building Docker Image"
	builder.Complete(app.ID, stageDesc, out, &err)
	return nil
}

// AuthToken retrieves the auth token for the given image.
func (b *Builder) AuthToken(ctx context.Context, app *builder.AppContext) (string, error) {
	dockerAuth, err := b.getACRDockerEntryFromARMToken(app.Ctx.Env.Registry)
	if err != nil {
		return "", err
	}
	buf, err := json.Marshal(dockerAuth)
	return base64.StdEncoding.EncodeToString(buf), err
}

func getRegistryName(registry string) string {
	return strings.TrimSuffix(registry, ".azurecr.io")
}

func blobComplete(metadata azblob.Metadata) bool {
	for k := range metadata {
		if strings.ToLower(k) == "complete" {
			return true
		}
	}
	return false
}

func (b *Builder) getACRDockerEntryFromARMToken(loginServer string) (*builder.DockerConfigEntryWithAuth, error) {
	accessToken := b.AdalToken.OAuthToken()

	directive, err := ReceiveChallengeFromLoginServer(loginServer)
	if err != nil {
		return nil, fmt.Errorf("failed to receive challenge: %s", err)
	}

	registryRefreshToken, err := PerformTokenExchange(
		loginServer, directive, b.Subscription.TenantID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to perform token exchange: %s", err)
	}

	glog.V(4).Infof("adding ACR docker config entry for: %s", loginServer)
	return &builder.DockerConfigEntryWithAuth{
		Username: DockerTokenLoginUsernameGUID,
		Password: registryRefreshToken,
	}, nil
}
