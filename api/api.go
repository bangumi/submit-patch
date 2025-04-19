package api

type WikiSubject struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	TypeID   int64    `json:"typeID"`
	Infobox  string   `json:"infobox"`
	Platform int      `json:"platform"`
	MetaTags []string `json:"metaTags"`
	Summary  string   `json:"summary"`
	Nsfw     bool     `json:"nsfw"`
}

type WikiEpisode struct {
	ID        int    `json:"id"`
	SubjectID int    `json:"subjectID"`
	Name      string `json:"name"`
	NameCN    string `json:"nameCN"`
	Type      int    `json:"type"`
	Ep        int    `json:"ep"`
	Duration  string `json:"duration"`
	Summary   string `json:"summary"`
	Disc      int    `json:"disc"`
	Date      string `json:"date"`
}
