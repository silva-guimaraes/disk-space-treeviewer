package main


import (
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
)

type dir struct {
    abs, Relative string
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

    // essa gambiarra toda é pra transformar uma lista de diretórios em uma árvore de diretórios já que
    // fs.WalkDir retorna um diretório de cada vez de maneira desestruturada.

    // realidade:
    // root/
    // ├─ bar/
    // │  ├─ foo/
    // │  │  ├─ barfoo/
    // │  │  │  ├─ foobar/
    // │  ├─ barfoo/
    // │  ├─ foofoo/
    // ├─ foo/
    // ├─ foobar/
    // │  ├─ foo/

    // fs.WalkDir:
    // root
    // root/bar
    // root/bar/foo
    // root/bar/foo/barfoo
    // root/bar/foo/barfoo/foobar
    // root/bar/barfoo
    // root/bar/foofoo
    // root/foo
    // root/foobar
    // root/foobar/foo

    // pra nossa sorte, fs.WalkDir retorna os diretórios em ordem alfabética, então, cada novo diretório introduzido
    // sempre será filho do diretório atual.


    // stack de diretórios. o ultimo elemento dessa lista é o diretório atual onde o tamanho dos arquivos e outros
    // diretórios subsequentes são salvos
    pwd := []*dir{}

    // faz com que o ultimo diretório passe a ser filho do diretório anterior
    // isso monta a nossa arvore ao final do fs.WalkDir
    popPwd := func() {
        pop := pwd[len(pwd)-1]
        pwd = pwd[:len(pwd)-1]

        current := pwd[len(pwd)-1]
        current.Children = append(current.Children, pop)

        // todo o tamanho que os filhos desse diretório possuem
        current.SizeDirs += pop.SizeDirs + pop.SizeFiles
    }

    // a cada novo diretório encontrado, pushar para a stack e tratar novo diretório como diretório atual.
    // apenas diretórios são salvos. cada arquivo encontrado tem o seu tamanho salvo no seu diretório parente.
    // se novo diretório não ter o mesmo parente que o diretório atual, isso significa que esses dois diretórios
    // são irmãos possivelmente, popPwd nesse caso.
    fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, fserr error) error {

        abs := filepath.Join(rootPath, path)

        // essa pasta não possue nada de importante e só me causa dor de cabeça.
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

    // capaz que tenham sobrado diretórios no pwd. isso transforma toda a arvore em um root apenas.
    for len(pwd) > 1 {
        popPwd()
    }

    // fim da gambiarra

    // formatar tamanho
    // formatted := float64(totalSize)
    // magnitude := []rune{'B', 'K', 'M', 'G'}
    // magnitudeIndex := 0
    // for formatted > 1000 && magnitudeIndex < len(magnitude)-1 {
    //     formatted /= 1000
    //     magnitudeIndex++
    // }

    t, err := template.New("foo").Parse(templ) 
    if err != nil {
        panic(err)
    }

    t.Execute(output, pwd[0])
    fmt.Println("saved to", outputFileName)
}
