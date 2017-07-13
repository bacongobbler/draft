var goImage = "golang:1.8"

// To set GOPATH correctly, we have to override the default
// path that Acid sets.
var localPath = "/go/src/github.com/Azure/draft";
var defaultGoEnv = {
  "DEST_PATH": localPath
};

var testJob = new Job("test");
testJob.image = goImage;
testJob.mountPath = localPath;
testJob.env = defaultGoEnv;
testJob.tasks = [
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

var dockerJob = new Job("docker");
dockerJob.image = "docker:17.05.0-ce-dind";
dockerJob.env = {
  "REGISTRY": "docker.io/",
  // TODO: change this back to microsoft once we are ready to ship
  "IMAGE_PREFIX": "bacongobbler",
  "DOCKER_DRIVER": "overlay"
}
dockerJob.tasks = [
  'dockerd --host=unix:///var/run/docker.sock &',
  'docker login -u="$DOCKER_USER" -p="$DOCKER_PASSWORD"',
  'make docker-build docker-push'
];
// run the docker job as a privileged container
dockerJob.container.securityContext = {
  privileged: true
}

events.push = function(e) {
  testJob.env["CODECOV_TOKEN"] = e.env.CODECOV_TOKEN;

  azureJob.env["AZURE_STORAGE_ACCOUNT"] = e.env.AZURE_STORAGE_ACCOUNT;
  azureJob.env["AZURE_STORAGE_CONTAINER"] = e.env.AZURE_STORAGE_CONTAINER;
  azureJob.env["AZURE_STORAGE_KEY"] = e.env.AZURE_STORAGE_KEY;
  azureJob.env["VERSION"] = e.commit;

  dockerJob.env["DOCKER_USER"] = e.env.DOCKER_USER;
  dockerJob.env["DOCKER_PASSWORD"] = e.env.DOCKER_PASSWORD;
  dockerJob.env["VERSION"] = e.commit;

  wg = new WaitGroup();
  wg.add(testJob);
  wg.add(azureJob);
  wg.add(dockerJob);

  wg.jobs.forEach(function (j) {
      j.background();
  })

  // HACK(bacongobbler): give the jobs a 10 minute headstart before we start to check in on them
  sleep(600);

  wg.wait();
}
