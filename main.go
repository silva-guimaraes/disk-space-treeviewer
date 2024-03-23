package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
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

    outputFileName := "output.html"
    output, err := os.Create(outputFileName)
    if err != nil {
        panic(err)
    }

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

    fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            panic(err)
        }
        info, err := d.Info()
        if err != nil {
            panic(err)
        }

        for abs, _ := filepath.Abs(path); len(pwd) > 0 && filepath.Dir(abs) != pwd[len(pwd)-1].abs; {
            popPwd()
        }

        if info.IsDir() {
            abs, _ := filepath.Abs(path)
            pwd = append(pwd, &dir{abs, path, nil, 0, 0})
        } else {
            counter++
            size := info.Size()
            totalSize += size
            pwd[len(pwd)-1].SizeFiles += size
        }

        // fmt.Print("\r", path)

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

    // fmt.Printf(
    //     "num of files: %d, total size: %.2f %cB, raw size: %d\n", counter, formatted,
    //     magnitude[magnitudeIndex], totalSize,
    // )
    // fmt.Println(pwd[0])
    // fmt.Println(pwd[0].SizeDirs + pwd[0].SizeFiles)

    // ret, err := json.Marshal(pwd[0])
    // if err != nil {
    //     panic(err)
    // }

    // fmt.Println(string(ret))

    t, err := template.New("foo").Parse(templ) 
    if err != nil {
        panic(err)
    }

    fmt.Println("saved to", outputFileName)
    t.Execute(output, pwd[0])
}
