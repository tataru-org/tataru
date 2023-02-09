package main

// everything is not implemented here because this bot doesn't need most of it

type XivMount struct {
	Name string `json:"name,omitempty"`
	Icon string `json:"icon,omitempty"`
}

type XivMinion struct {
	Name string `json:"name,omitempty"`
	Icon string `json:"icon,omitempty"`
}

type XivReducedCharacterProfile struct {
	Avatar       string `json:"avatar,omitempty"`
	FeastMatches int    `json:"feast_matches,omitempty"`
	ID           uint   `json:"id,omitempty"`
	Lang         string `json:"lang,omitempty"`
	Name         string `json:"name,omitempty"`
	Rank         string `json:"rank,omitempty"`
	RankIcon     string `json:"rank_icon,omitempty"`
	Server       string `json:"server,omitempty"`
}

type XivCharacterProfile struct {
	Name string `json:"name,omitempty"`
	ID   uint   `json:"id,omitempty"`
}

type XivPagination struct {
	Page           int `json:"page,omitempty"`
	PageNext       int `json:"page_next,omitempty"`
	PagePrev       int `json:"page_prev,omitempty"`
	PageTotal      int `json:"page_total,omitempty"`
	Results        int `json:"results,omitempty"`
	ResultsPerPage int `json:"results_per_page,omitempty"`
	ResultsTotal   int `json:"results_total,omitempty"`
}

type XivCharacterSearch struct {
	Pagination XivPagination                `json:"pagination,omitempty"`
	Results    []XivReducedCharacterProfile `json:"results,omitempty"`
}

type XivCharacter struct {
	Character          XivCharacterProfile          `json:"character,omitempty"`
	FreeCompanyMembers []XivReducedCharacterProfile `json:"free_company_members,omitempty"`
	Minions            []XivMinion                  `json:"minions,omitempty"`
	Mounts             []XivMount                   `json:"mounts,omitempty"`
}
