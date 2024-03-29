package main

import (
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	// "strings"
)

type dir struct {
    abs, Relative string
    // stat                fs.FileInfo
    Children            []*dir
    SizeFiles, SizeDirs int64
}

//go:embed template.html
var templ string

func main() {
    
    if len(os.Args) < 2 {
        panic(os.Args)
    }

    rootPath, err := filepath.Abs(os.Args[1])
    if err != nil {
        panic(err)
    }
    fileSystem := os.DirFS(rootPath)

    here, err := os.Getwd()
    if err != nil {
        panic(err)
    }

    outputFileName := "output.html"
    output, err := os.Create(outputFileName)
    if err != nil {
        if errors.Is(err, fs.ErrPermission) {
            fmt.Fprintf(os.Stderr, "can't create file here: '%s'\n", here)
            os.Exit(1)
        }
        panic(err)
    }
    defer output.Close()

    counter := 0
    totalSize := int64(0)
    pwd := []*dir{}

    popPwd := func() {
        pop := pwd[len(pwd)-1]
        pwd = pwd[:len(pwd)-1]

        current := pwd[len(pwd)-1]
        current.Children = append(current.Children, pop)
        current.SizeDirs += pop.SizeDirs + pop.SizeFiles
    }

    fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, fserr error) error {

        abs := filepath.Join(rootPath, path)

        if abs == "/proc" {
            fmt.Println(abs)
            fmt.Println("problematic folder, skipping:", "/proc")
            return fs.SkipDir
        }

        if fserr != nil {
            permissionDenied := errors.Is(fserr, fs.ErrPermission)

            if permissionDenied && d.IsDir() {
                fmt.Println("permission denied, skipping directory:", path)
                return fs.SkipDir;

            } else if permissionDenied {
                fmt.Println("permission denied:", path)
                return nil;

            } else {
                panic(fserr)
            }
        }
        info, err := d.Info()
        if err != nil {
            if errors.Is(err, fs.ErrNotExist) {
                fmt.Println("file no longer exists:", abs)
                return nil
            }
            panic(err)
        }

        for len(pwd) > 0 && filepath.Dir(abs) != pwd[len(pwd)-1].abs {
            popPwd()
        }

        if info.IsDir() {
            pwd = append(pwd, &dir{abs, path, nil, 0, 0})
        } else {
            counter++
            size := info.Size()
            totalSize += size
            pwd[len(pwd)-1].SizeFiles += size
        }

        return nil
    })


    for len(pwd) > 1 {
        popPwd()
    }

    formatted := float64(totalSize)
    magnitude := []rune{'B', 'K', 'M', 'G'}
    magnitudeIndex := 0

    for formatted > 1000 && magnitudeIndex < len(magnitude)-1 {
        formatted /= 1000
        magnitudeIndex++
    }

    t, err := template.New("foo").Parse(templ) 
    if err != nil {
        panic(err)
    }

    t.Execute(output, pwd[0])
    fmt.Println("saved to", outputFileName)
}
