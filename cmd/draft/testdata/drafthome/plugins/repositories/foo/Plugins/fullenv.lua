local name = "fullenv"
local version = "0.1.0"

plugin = {
    name = name,
    short = "show env vars",
    description = "show all env vars",
    homepage = "",
    version = version,
    useTunnel = false,
    packages = {
        {
            os = "darwin",
            arch = "amd64",
            url = "",
            sha256 = "",
            path = name .. ".sh",
        },
        {
            os = "linux",
            arch = "amd64",
            url = "",
            sha256 = "",
            path = name .. ".sh",
        },{
            os = "windows",
            arch = "amd64",
            url = "",
            sha256 = "",
            path = name .. ".sh",
        },
    }
}
