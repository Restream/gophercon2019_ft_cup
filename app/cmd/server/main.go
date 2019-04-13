package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/buaazp/fasthttprouter"
	"github.com/restream/reindexer"
	_ "github.com/restream/reindexer/bindings/builtin"
	"github.com/valyala/fasthttp"
)

var (
	// db
	db *reindexer.Reindexer

	// http
	headerContentType              = []byte("Content-Type")
	headerApplicationJSON          = []byte("application/json")
	headerAccessControlAllowOrigin = []byte("Access-Control-Allow-Origin")
	headerAllowAll                 = []byte("*")

	// def
	emptyIntSlice    = make([]int, 0)
	emptyStringSlice = make([]string, 0)
)

func main() {
	// time
	start := time.Now()

	// init
	log.Println("Create indexes")

	db = reindexer.NewReindex("builtin:///tmp/reindex/gopherconf")
	err := db.OpenNamespace("media_items", reindexer.DefaultNamespaceOptions(), &MediaItem{})
	if err != nil {
		log.Fatalln("Cant open namespace", err)
	}

	err = db.OpenNamespace("epgs", reindexer.DefaultNamespaceOptions(), &Epg{})
	if err != nil {
		log.Fatalln("Cant open namespace", err)
	}

	miID := reindexer.DefaultFtFastConfig()
	// TODO: Enable this
	//miID.MaxTyposInWord = 1
	//miID.EnableTranslit = true
	//miID.EnableKbLayout = true

	if err := db.ConfigureIndex("media_items", "search", miID); err != nil {
		log.Fatalln("Cant open namespace", err)
	}

	if err := db.ConfigureIndex("epgs", "search", miID); err != nil {
		log.Fatalln("Cant open namespace", err)
	}

	var path = "/data"
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		switch f.Name() {
		case "media_items.json":
			updateMediaItemsFromFile(path + "/" + f.Name())
		case "epg.json":
			updateEpgFromFile(path + "/" + f.Name())
		default:
			continue
		}

		log.Println("Process file", f.Name())
	}

	log.Println("Elapsed", time.Now().Sub(start))

	// API
	router := fasthttprouter.New()
	router.GET("/", indexHandler)
	router.GET("/api/v1/search", searchHandler)
	router.GET("/api/v1/media_items", mediaItemsHandler)
	router.GET("/api/v1/epg", epgHandler)

	if err := fasthttp.ListenAndServe(":8080", router.Handler); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func updateMediaItemsFromFile(filePath string) {
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("Error read file %s: %s\n", filePath, err.Error())
	}

	films := make(MediaItems, 0, 256)
	err = json.Unmarshal(raw, &films)
	if err != nil {
		log.Printf("Error parse file %s: %s\n", filePath, err.Error())
	}

	for _, film := range films {
		err := db.Upsert("media_items", film)
		if err != nil {
			log.Printf("Error upsert film from file %s: %s\n", filePath, err.Error())
		}
	}
}

func updateEpgFromFile(filePath string) {
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("Error read file %s: %s\n", filePath, err.Error())
	}

	films := make(Epgs, 0, 256)
	err = json.Unmarshal(raw, &films)
	if err != nil {
		log.Printf("Error parse file %s: %s\n", filePath, err.Error())
	}

	for _, film := range films {
		err := db.Upsert("epgs", film)
		if err != nil {
			log.Printf("Error upsert film from file %s: %s\n", filePath, err.Error())
		}
	}
}

func indexHandler(ctx *fasthttp.RequestCtx) {
	_, _ = fmt.Fprintf(ctx, "Oops!")
}

func searchHandler(ctx *fasthttp.RequestCtx) {
	var (
		limit  = ctx.QueryArgs().GetUintOrZero("limit")
		offset = ctx.QueryArgs().GetUintOrZero("offset")
		query  = string(ctx.QueryArgs().Peek("query"))
	)

	it := db.Query("media_items").ReqTotal().Limit(limit).Offset(offset).Match("search", query).Exec()
	if it.Error() != nil {
		log.Println(it.Error())
	}
	defer it.Close()

	var res = SearchResponse{
		TotalItems: it.Count(),
		Items:      make([]*SearchItem, 0, it.Count()),
	}
	for it.Next() {
		item := it.Object().(*MediaItem)
		setMediaItemDefaults(item)
		res.Items = append(res.Items, &SearchItem{Type: "media_item", MediaItem: item})
	}

	if err := json.NewEncoder(ctx).Encode(&res); err != nil {
		log.Println(err)
	}

	ctx.Response.Header.SetCanonical(headerContentType, headerApplicationJSON)
	ctx.Response.Header.SetCanonical(headerAccessControlAllowOrigin, headerAllowAll)
}

func mediaItemsHandler(ctx *fasthttp.RequestCtx) {
	var (
		limit   = ctx.QueryArgs().GetUintOrZero("limit")
		offset  = ctx.QueryArgs().GetUintOrZero("offset")
		yearGe  = ctx.QueryArgs().GetUintOrZero("year_ge")
		yearLe  = ctx.QueryArgs().GetUintOrZero("year_le")
		sortBy  = string(ctx.QueryArgs().Peek("sort_by"))
		sortDir = string(ctx.QueryArgs().Peek("sort_dir"))
	)

	q := db.Query("media_items").
		ReqTotal().
		Limit(limit).
		Offset(offset).
		Sort("search", sortDir == "desc", sortBy)

	if yearGe > 0 {
		q = q.Where("search", reindexer.GE, yearGe)
	}

	if yearLe > 0 {
		q = q.Where("search", reindexer.LE, yearLe)
	}

	if len(sortBy) == 0 {
		sortBy = "name"
	}

	it := q.Exec()
	if it.Error() != nil {
		log.Println(it.Error())
	}
	defer it.Close()

	var res = MediaItemResponse{
		TotalItems: it.Count(),
		Items:      make(MediaItems, 0, it.Count()),
	}
	for it.Next() {
		item := it.Object().(*MediaItem)
		setMediaItemDefaults(item)
		res.Items = append(res.Items, item)
	}

	if err := json.NewEncoder(ctx).Encode(&res); err != nil {
		log.Println(err)
	}

	ctx.Response.Header.SetCanonical(headerContentType, headerApplicationJSON)
	ctx.Response.Header.SetCanonical(headerAccessControlAllowOrigin, headerAllowAll)
}

func epgHandler(ctx *fasthttp.RequestCtx) {
	var (
		limit     = ctx.QueryArgs().GetUintOrZero("limit")
		offset    = ctx.QueryArgs().GetUintOrZero("offset")
		startTime = ctx.QueryArgs().GetUintOrZero("start_time")
		endTime   = ctx.QueryArgs().GetUintOrZero("end_time")
	)

	q := db.Query("epgs").
		ReqTotal().
		Limit(limit).
		Offset(offset)

	if startTime > 0 {
		q = q.Where("start_time", reindexer.GE, startTime)
	}

	if endTime > 0 {
		q = q.Where("end_time", reindexer.LE, endTime)
	}

	it := q.Exec()
	if it.Error() != nil {
		log.Println(it.Error())
	}
	defer it.Close()

	var res = EpgsResponse{
		TotalItems: it.Count(),
		Items:      make(Epgs, 0, it.Count()),
	}
	for it.Next() {
		item := it.Object().(*Epg)
		res.Items = append(res.Items, item)
	}

	if err := json.NewEncoder(ctx).Encode(&res); err != nil {
		log.Println(err)
	}

	ctx.Response.Header.SetCanonical(headerContentType, headerApplicationJSON)
	ctx.Response.Header.SetCanonical(headerAccessControlAllowOrigin, headerAllowAll)
}

func setMediaItemDefaults(item *MediaItem) {
	if item.Countries == nil {
		item.Countries = emptyStringSlice
	}

	if item.Genres == nil {
		item.Genres = emptyStringSlice
	}

	if item.Packages == nil {
		item.Packages = emptyIntSlice
	}

	if item.Packages == nil {
		item.Packages = emptyIntSlice
	}
}
