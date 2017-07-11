events.push = function(e) {
  var registries = {
    "dockerhub": {
      "production": {
        "name": "microsoft",
        "email": "matt.fisher@microsoft.com",
        "username": "bacongobbler",
        "password": e.env.dockerPasswd
      }
    }
  }

  // This is a Go project, so we want to set it up for Go.
  var gopath = "/go";

  // To set GOPATH correctly, we have to override the default
  // path that Acid sets.
  var localPath = gopath + "/src/github.com/Azure/draft";

  var goBuild = new Job("draft-test");

  goBuild.image = "golang:1.8";
  goBuild.mountPath = localPath

  // Set a few environment variables.
  goBuild.env = {
      "DEST_PATH": localPath,
      "GOPATH": gopath,
      "CODECOV_TOKEN": e.env.codecovToken
  };

  goBuild.tasks = [
    "cd $DEST_PATH",
    "make bootstrap",
    "make build",
    "make test",
    "curl -s https://codecov.io/bash | bash -s - -t $CODECOV_TOKEN"
  ];

  goBuild.run();
}
