package types

type BuildMessage struct {
	Event         string `json:"event"`
	Build         Build  `json:"build"`
	RepoId        string `json:"repoId"`
	ReceiptHandle string `json:"-"`
}

type Build struct {
	Result         string `json:"result"`
	Status         string `json:"status"`
	WebURL         string `json:"webUrl"`
	Commit         string `json:"commit"`
	CommitterName  string `json:"committerName"`
	Branch         string `json:"branch"`
	Number         int    `json:"number"`
	StorybookURL   string `json:"storybookUrl"`
	ChangeCount    int    `json:"changeCount"`
	ComponentCount int    `json:"componentCount"`
	SpecCount      int    `json:"specCount"`
}
