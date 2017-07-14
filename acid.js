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
testJob.env = {
  "DEST_PATH": localPath,
  "CODECOV_TOKEN": project.secrets.CODECOV_TOKEN
};
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
azureJob.env = {
  "DEST_PATH": localPath,
  "AZURE_STORAGE_ACCOUNT": project.secrets.AZURE_STORAGE_ACCOUNT,
  "AZURE_STORAGE_CONTAINER": project.secrets.AZURE_STORAGE_CONTAINER,
  "AZURE_STORAGE_KEY": project.secrets.AZURE_STORAGE_KEY
};
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
dockerJob.mountPath = localPath;
dockerJob.env = {
  "DEST_PATH": localPath,
  "GOPATH": "/go",
  "PATH": "/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
  "REGISTRY": "docker.io/",
  // TODO: change this back to microsoft once we are ready to ship
  "IMAGE_PREFIX": "bacongobbler",
  "DOCKER_DRIVER": "overlay",
  "DOCKER_USER": project.secrets.DOCKER_USER,
  "DOCKER_PASSWORD": project.secrets.DOCKER_PASSWORD
}
dockerJob.tasks = [
  'apk add --no-cache bash git go libc-dev make nodejs',
  'cd $DEST_PATH',
  'make bootstrap',
  'dockerd --host=unix:///var/run/docker.sock &',
  'make docker-build',
  'docker login -u="$DOCKER_USER" -p="$DOCKER_PASSWORD"',
  'make docker-push'
];
// run the docker job as a privileged container
dockerJob.container.securityContext = {
  privileged: true
}

events.push = function(e) {
  azureJob.env["VERSION"] = e.commit;
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
