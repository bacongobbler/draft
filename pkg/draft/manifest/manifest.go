package manifest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/technosophos/moniker"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"github.com/Azure/draft/pkg/draft/tunnel"
	pkgio "github.com/Azure/draft/pkg/io"
	"github.com/Azure/draft/pkg/kube/podutil"
)

const (
	// DefaultEnvironmentName is the name invoked from draft.toml on `draft up` when
	// --environment is not supplied.
	DefaultEnvironmentName = "development"
	// DefaultNamespace specifies the namespace apps should be deployed to by default.
	DefaultNamespace = "default"
	// DefaultWatchDelaySeconds is the time delay between files being changed and when a
	// new draft up` invocation is called when --watch is supplied.
	DefaultWatchDelaySeconds = 2
	// LabelKey is the label selector key on a pod that allows
	// us to identify which draft app a pod is associated with
	LabelKey = "draft"
	// ControllerLabelKey is the label selector key on a pod that allows
	// us to identify which controller a pod is associated with
	ControllerLabelKey = "controller"
	// BuildIDKey is the label selector key on a pod that specifies
	// the build ID of the application
	BuildIDKey = "buildID"
)

// Manifest represents a draft.toml
type Manifest struct {
	Environments map[string]*Environment `toml:"environments"`
}

// Environment represents the environment for a given app at build time
type Environment struct {
	Name              string   `toml:"name,omitempty"`
	ContainerBuilder  string   `toml:"container-builder,omitempty"`
	Registry          string   `toml:"registry,omitempty"`
	ResourceGroupName string   `toml:"resource-group-name,omitempty"`
	Namespace         string   `toml:"namespace,omitempty"`
	Values            []string `toml:"set,omitempty"`
	Wait              bool     `toml:"wait"`
	Watch             bool     `toml:"watch"`
	WatchDelay        int      `toml:"watch-delay,omitempty"`
	OverridePorts     []string `toml:"override-ports,omitempty"`
	AutoConnect       bool     `toml:"auto-connect"`
	CustomTags        []string `toml:"custom-tags,omitempty"`
	Controllers       []string `toml:"controllers"`
	Chart             string   `toml:"chart"`
}

// New creates a new manifest with the Environments intialized.
func New() *Manifest {
	m := Manifest{
		Environments: make(map[string]*Environment),
	}
	m.Environments[DefaultEnvironmentName] = &Environment{
		Name:        generateName(),
		Namespace:   DefaultNamespace,
		Wait:        true,
		Watch:       false,
		WatchDelay:  DefaultWatchDelaySeconds,
		AutoConnect: false,
	}
	return &m
}

// generateName generates a name based on the current working directory or a random name.
func generateName() string {
	var name string
	cwd, err := os.Getwd()
	if err == nil {
		name = filepath.Base(cwd)
	} else {
		namer := moniker.New()
		name = namer.NameSep("-")
	}
	return name
}

// Connect tunnels to a Kubernetes pod running the application and returns the connection information
func (e *Environment) Connect(clientset kubernetes.Interface, clientConfig *restclient.Config, targetContainer string, overridePorts []string, buildID string) (*Connection, error) {
	var cc []*ContainerConnection

	for _, controller := range e.Controllers {
		pod, err := podutil.GetPod(e.Namespace, LabelKey, e.Name, ControllerLabelKey, controller, BuildIDKey, buildID, clientset)
		if err != nil {
			return nil, err
		}
		m, err := getPortMapping(overridePorts)
		if err != nil {
			return nil, err
		}

		// if no container was specified as flag, return tunnels to all containers in pod
		if targetContainer == "" {
			for _, c := range pod.Spec.Containers {
				var tt []*tunnel.Tunnel

				// iterate through all ports of the container and create tunnels
				for _, p := range c.Ports {
					remote := int(p.ContainerPort)
					local := m[remote]
					t := tunnel.NewWithLocalTunnel(clientset.CoreV1().RESTClient(), clientConfig, e.Namespace, pod.Name, remote, local)
					tt = append(tt, t)
				}
				cc = append(cc, &ContainerConnection{
					PodName:       pod.Name,
					ContainerName: c.Name,
					Tunnels:       tt,
				})
			}
		} else {
			var tt []*tunnel.Tunnel

			// a container was specified - return tunnel to specified container
			ports, err := getTargetContainerPorts(pod.Spec.Containers, targetContainer)
			if err != nil {
				return nil, err
			}

			// iterate through all ports of the container and create tunnels
			for _, p := range ports {
				local := m[p]
				t := tunnel.NewWithLocalTunnel(clientset.CoreV1().RESTClient(), clientConfig, e.Namespace, pod.Name, p, local)
				tt = append(tt, t)
			}

			cc = append(cc, &ContainerConnection{
				PodName:       pod.Name,
				ContainerName: targetContainer,
				Tunnels:       tt,
			})
		}
	}

	return &Connection{
		ContainerConnections: cc,
		Clientset:            clientset,
	}, nil
}

// Connection encapsulated information to connect to an application
type Connection struct {
	ContainerConnections []*ContainerConnection
	Clientset            kubernetes.Interface
}

// ContainerConnection encapsulates a connection to a container in a pod
type ContainerConnection struct {
	Tunnels       []*tunnel.Tunnel
	ContainerName string
	PodName       string
}

// DeployedApplication returns deployment information about the deployed instance
//  of the source code given a path to your draft.toml file and the name of the
//  draft environment
func DeployedApplication(draftTomlPath, draftEnvironment string) (*Environment, error) {
	var draftConfig Manifest
	if _, err := toml.DecodeFile(draftTomlPath, &draftConfig); err != nil {
		return nil, err
	}

	appConfig, found := draftConfig.Environments[draftEnvironment]
	if !found {
		return nil, fmt.Errorf("Environment %v not found", draftEnvironment)
	}

	return appConfig, nil
}

func getPortMapping(overridePorts []string) (map[int]int, error) {
	var portMapping = make(map[int]int, len(overridePorts))

	for _, p := range overridePorts {
		m := strings.Split(p, ":")
		local, err := strconv.Atoi(m[0])
		if err != nil {
			return nil, fmt.Errorf("cannot get port mapping: %v", err)
		}

		remote, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("cannot get port mapping: %v", err)
		}

		// check if remote port already exists in port mapping
		_, exists := portMapping[remote]
		if exists {
			return nil, fmt.Errorf("remote port %v already mapped", remote)
		}

		// check if local port already exists in port mapping
		for _, l := range portMapping {
			if local == l {
				return nil, fmt.Errorf("local port %v already mapped", local)
			}
		}

		portMapping[remote] = local
	}

	return portMapping, nil
}

// RequestLogStream returns a stream of the application pod's logs
func (c *Connection) RequestLogStream(namespace string, logLines int64) (io.ReadCloser, error) {
	var streams []io.ReadCloser
	for _, containerConnection := range c.ContainerConnections {
		req := c.Clientset.CoreV1().Pods(namespace).GetLogs(containerConnection.PodName,
			&v1.PodLogOptions{
				Follow:    true,
				TailLines: &logLines,
				Container: containerConnection.ContainerName,
			})

		s, err := req.Stream()
		if err != nil {
			for _, stream := range streams {
				stream.Close()
			}
			return nil, err
		}
		streams = append(streams, s)
	}

	return pkgio.MultiReadCloser(streams...), nil

}

func getTargetContainerPorts(containers []v1.Container, targetContainer string) ([]int, error) {
	var ports []int
	containerFound := false

	for _, c := range containers {

		if c.Name == targetContainer && !containerFound {
			containerFound = true
			for _, p := range c.Ports {
				ports = append(ports, int(p.ContainerPort))
			}
		}
	}

	if containerFound == false {
		return nil, fmt.Errorf("container '%s' not found", targetContainer)
	}

	return ports, nil
}

func (e *Environment) GetPodNames(buildID string, clientset kubernetes.Interface) ([]string, error) {
	label := map[string]string{LabelKey: e.Name}
	annotations := map[string]string{BuildIDKey: buildID}
	return podutil.ListPodNames(e.Namespace, label, annotations, clientset)
}
