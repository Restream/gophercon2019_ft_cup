package main

//easyjson:json
type (
	Person struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	Persons []*Person

	MediaItem struct {
		ID          int      `reindex:"id,,pk" json:"id"`
		Name        string   `reindex:"name" json:"name"`
		Type        string   `reindex:"type" json:"type"`
		Duration    int      `reindex:"duration," json:"duration"`
		Countries   []string `reindex:"countries" json:"countries"`
		AgeValue    int      `reindex:"age_value" json:"age_value"`
		Year        string   `reindex:"year" json:"year"`
		Logo        string   `reindex:"logo" json:"logo"`
		Rating      float64  `reindex:"rating" json:"rating"`
		Description string   `reindex:"description" json:"description"`
		Genres      []string `reindex:"genres" json:"genres"`
		Persons     Persons  `reindex:"persons" son:"persons"`
		Packages    []int    `reindex:"packages" json:"packages"`
		AssetTypes  []int    `reindex:"asset_types" json:"asset_types"`
		_           struct{} `reindex:"name+countries+persons.name+genres+year+description=search,text,composite,dense"`
	}

	MediaItems []*MediaItem

	Channel struct {
		ID   int    `reindex:"id" json:"id"`
		Name string `reindex:"name" json:"name"`
		Logo string `reindex:"logo" json:"logo"`
	}

	Epg struct {
		ID          int      `reindex:"id,,pk" json:"id"`
		Name        string   `reindex:"name" json:"name"`
		AgeValue    int      `reindex:"age_value" json:"age_value"`
		StartTime   int      `reindex:"start_time" json:"start_time"`
		EndTime     int      `reindex:"end_time" json:"end_time"`
		Genre       string   `reindex:"genre" json:"genre"`
		Description string   `reindex:"description" json:"description"`
		Logo        string   `reindex:"logo" json:"logo"`
		Channel     *Channel `reindex:"channel" json:"channel"`
		LocationID  int      `reindex:"location_id" json:"location_id"`
		_           struct{} `reindex:"name+genre+channel.name+description=search,text,composite,dense"`
	}

	Epgs []*Epg

	SearchItem struct {
		Type      string     `json:"type"`
		MediaItem *MediaItem `json:"media_item,omitempty"`
		Epg       *Epg       `json:"epg,omitempty"`
	}

	SearchResponse struct {
		TotalItems int           `json:"total_items"`
		Items      []*SearchItem `json:"items"`
	}

	MediaItemResponse struct {
		TotalItems int        `json:"total_items"`
		Items      MediaItems `json:"items"`
	}

	EpgsResponse struct {
		TotalItems int  `json:"total_items"`
		Items      Epgs `json:"items"`
	}
)
