package main

import (
    "bytes"
    "github.com/attic-labs/noms/ref"
)

// Main data model
type Photo struct {
    NomsName string          `noms:"$name"`
    Height   string          `noms:"height"`
    Id       string          `noms:"id"`
    Image    *bytes.Reader   `noms:"image"`
    Tags     map[string]bool `noms:"tags"`
    Title    string          `noms:"title"`
    Url      string          `noms:"url"`
    Width    string          `noms:"width"`
}

type Album struct {
    Id        string
    Title     string
    NumPhotos int
    Photos    []Photo
}

type User struct {
    Id     string
    Name   string
    Albums []Album
}

// Types used for communicating with Go routines that fetch photos
type PhotoMessage struct {
    Index int
    Photo Photo
}

type RefMessage struct {
    Index int
    Ref ref.Ref
}

// Used for sorting RefMessages by index field
type ByIndex []RefMessage

func (slice ByIndex) Len() int {
    return len(slice)
}

func (slice ByIndex) Less(i, j int) bool {
    return slice[i].Index < slice[j].Index;
}

func (slice ByIndex) Swap(i, j int) {
    slice[i], slice[j] = slice[j], slice[i]
}

// Used for unmarshalling json 
type AlbumJson struct {
    Feed struct {
        UserName struct {
            V string `json:"$t"`
        } `json:"gphoto$nickname"`
        Id struct {
            V string `json:"$t"`
        } `json:"gphoto$id"`
            NumPhotos struct {
        V int `json:"$t"`
            } `json:"gphoto$numphotos"`
        Title struct {
            V string `json:"$t"`
        }
        UserId struct {
            V string `json:"$t"`
        } `json:"gphoto$user"`
        Entry []struct {
            Content struct {
                Src  string
                Type string
            }
            Height struct {
                V string `json:"$t"`
            } `json:"gphoto$height"`
            Id struct {
                V string `json:"$t"`
            } `json:"gphoto$id"`
            Size struct {
                V string `json:"$t"`
            } `json:"gphoto$size"`
            MediaGroup struct {
                Tags struct {
                    V string `json:"$t"`
                } `json:"media$keywords"`
            } `json:"media$group"`
            Timestamp struct {
                V string `json:"$t"`
            } `json:"gphoto$timestamp"`
            Title struct {
                V string `json:"$t"`
            }
            Width struct {
                V string `json:"$t"`
            } `json:"gphoto$width"`
        }
    }
}

type AlbumListJson struct {
    Feed struct {
        UserName struct {
            V string `json:"$t"`
        } `json:"gphoto$nickname"`
        Entry []struct {
            Id struct {
                V string `json:"$t"`
            } `json:"gphoto$id"`
            NumPhotos struct {
                V int `json:"$t"`
            } `json:"gphoto$numphotos"`
            Title struct {
                V string `json:"$t"`
            }
        }
        UserId struct {
            V string `json:"$t"`
        } `json:"gphoto$user"`
    }
}

type RefreshTokenJson struct {
    RefreshToken string
}
