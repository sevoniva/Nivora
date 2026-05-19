package version

type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

var (
	Version = "0.9.0-beta-candidate"
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
