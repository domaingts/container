package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    5 * time.Second,
			DisableCompression: true,
		},
	}
	cv := "1.3.0"
	url := fmt.Sprintf("https://github.com/containernetworking/plugins/releases/download/v%s/cni-plugins-linux-amd64-v%s.tgz", cv, cv)
	err := unzip(client, url, "/opt/cni/bin")
	if err != nil {
		panic(err)
	}
	nv := "1.7.0"
	url = fmt.Sprintf("https://github.com/containerd/nerdctl/releases/download/v%s/nerdctl-%s-linux-amd64.tar.gz", nv, nv)
	err = unzip(client, url, "/usr/local/bin")
	if err != nil {
		panic(err)
	}
	// err = removeDir("test")
	// if err != nil {
	// 	panic(err)
	// }
}

func unzip(client *http.Client, url string, path string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = mkDir(path, 0600)
	if err != nil {
		return err
	}
	gzf, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gzf)
	defer func() {
		_ = gzf.Close()
	}()
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		fmt.Println("name", hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			fmt.Println("Directory", hdr.Name)
			err = mkDir(hdr.Name, hdr.Mode)
			if err != nil {
				return err
			}
		default:
			err = write2File(filepath.Join(path, hdr.Name), hdr.Mode, tr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}

func mkDir(path string, mode int64) error {
	fmt.Println("make dir", path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		parent := substr(path, 0, strings.LastIndex(path, "/"))
		if _, err = os.Stat(parent); os.IsNotExist(err){
			err = mkDir(parent, mode)
			if err != nil {
				return err
			}
		}
		err = os.Mkdir(path, fs.FileMode(uint32(mode)))
		if err != nil {
			return err
		}
	}	
	return nil
}

func write2File(path string, mode int64, reader io.Reader) error {
	err := mkDir(filepath.Dir(path), 0600)
	if err != nil {
		return err
	}
	inner, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, fs.FileMode(uint32(mode)))
	if err != nil {
		return err
	}
	defer inner.Close()
	if _, err := io.Copy(inner, reader); err != nil {
		return err
	}
	defer inner.Sync()
	return nil
}

func removeDir(path string) error {
	return os.RemoveAll(path)
}
