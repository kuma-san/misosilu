package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "time"
    "math/rand"
    "strconv"

    "github.com/zenazn/goji"
    "github.com/zenazn/goji/web"
    "github.com/garyburd/redigo/redis"
)

type ShortenSuccess struct {
    Shorturl    string
    Created     string
}

type ShortenError struct {
    Error       string
    Message     string
}

func root(c web.C, w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "this is root")
}

func shorten(c web.C, w http.ResponseWriter, r *http.Request) {
    prefix := c.URLParams["prefix"]

//init redis
    rc, err := redis.Dial("tcp", ":6379")
    if err != nil {
       panic(err)
    }
    defer rc.Close()

    longUrl, err := redis.String(rc.Do("HGET", prefix, "longUrl"))
    if err != nil {
        fmt.Fprintf(w, "no such suffix")
    } else {
        count, err := redis.String(rc.Do("HGET", prefix, "count"))
        if err != nil {
            count = "0"
        }
        a, _ := strconv.Atoi(count)
        count = fmt.Sprint(a + 1)
        rc.Do("HSET", prefix, "count", count)
        http.Redirect(w, r, longUrl, 308)
    }
}

func apiShorten(c web.C, w http.ResponseWriter, r *http.Request) {
//init redis
    rc, err := redis.Dial("tcp", ":6379")
    if err != nil {
       panic(err)
    }
    defer rc.Close()

//init params
    longUrl, err := url.QueryUnescape(r.FormValue("url"))
    if err != nil {
        log.Fatal(err)
    }
    prefix := r.FormValue("prefix")
    //key := r.FormValue("key")

/* does not check key yet
    if key 
*/

    if prefix == "" {
        prefix = randSeq(8)
    }

//check if prefix exists
    _, err = redis.String(rc.Do("HGET", prefix, "longUrl"))
    if err == nil {
        //already in use
        result := ShortenError{"error", "prefix already in use"}
        j, _ := json.Marshal(result)
        fmt.Fprintf(w, string(j))
    } else {
        rc.Do("HSET", prefix, "longUrl", longUrl)
        rc.Do("HSET", prefix, "count", "0")
        rc.Do("HSET", prefix, "created", int32(time.Now().Unix()))
        rc.Do("HSET", prefix, "ip", r.RemoteAddr)

        created, err := redis.String(rc.Do("HGET", prefix, "created"))
        if err != nil {
            log.Fatal("cannot find created")
            Result := ShortenError{"error", "cannot saved data to redis."}
            j, _ := json.Marshal(Result)
            fmt.Fprintf(w, string(j))
        } else {
            Result := ShortenSuccess{"https://misosi.lu/" + prefix, created}
            j, _ := json.Marshal(Result)
            fmt.Fprintf(w, string(j))
        }
    }
}

func randSeq(n int) string {
    letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
    b := make([]rune, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}

func main() {
    goji.Get("/", root)
    goji.Post("/api/v1/shorten/", apiShorten)
    goji.Get("/:prefix", shorten)
    goji.Serve()
}
