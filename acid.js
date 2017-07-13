var goImage = "golang:1.8"

// To set GOPATH correctly, we have to override the default
// path that Acid sets.
var localPath = "/go/src/github.com/Azure/draft";
var defaultGoEnv = {
  "DEST_PATH": localPath
};

var azure = {
  "container": "draft",
  "storageAccount": "azuredraft",
}

var registries = {
  "dockerhub": {
    "production": {
      "name": "microsoft",
      "email": "matt.fisher@microsoft.com",
      "username": "bacongobbler"
    }
  }
}

var buildJob = new Job("test");
buildJob.image = goImage;
buildJob.mountPath = localPath;
buildJob.env = defaultGoEnv;
buildJob.tasks = [
  'cd $DEST_PATH',
  'make bootstrap',
  'make test',
  'curl -s https://codecov.io/bash | bash -s - -t $CODECOV_TOKEN'
];

// if the build succeeds, let's push up some artifacts
var azureJob = new Job("azure");
azureJob.image = goImage;
azureJob.mountPath = localPath;
azureJob.env = defaultGoEnv;

azureJob.tasks = [
  // install azure-cli
  'apt-get update -y',
  'apt-get install -y apt-transport-https',
  'echo "deb [arch=amd64] https://packages.microsoft.com/repos/azure-cli/ wheezy main" | tee /etc/apt/sources.list.d/azure-cli.list',
  'apt-key adv --keyserver packages.microsoft.com --recv-keys 417A0893',
  'apt-get update -y',
  'apt-get install -y azure-cli',
  'cd $DEST_PATH',
  'make bootstrap',
  'make build-cross',
  'make dist checksum',
  'az storage blob upload-batch --source _dist/ --destination $AZURE_STORAGE_CONTAINER --pattern *.tar.gz*'
];

events.push = function(e) {
  buildJob.env["CODECOV_TOKEN"] = e.env.CODECOV_TOKEN;

  azureJob.env["AZURE_STORAGE_ACCOUNT"] = e.env.AZURE_STORAGE_ACCOUNT;
  azureJob.env["AZURE_STORAGE_CONTAINER"] = e.env.AZURE_STORAGE_CONTAINER;
  azureJob.env["AZURE_STORAGE_KEY"] = e.env.AZURE_STORAGE_KEY;
  azureJob.env["VERSION"] = e.commit;

  wg = new WaitGroup();
  wg.add(buildJob);
  wg.add(azureJob);

  wg.run();
}
