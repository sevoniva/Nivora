package version

type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

var (
	Version = "0.1.0-alpha.1"
	Commit  = "unknown"
	Date    = "unknown"
)

func Current() Info {
	return Info{
		Name:    "nivora",
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}
}
