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
