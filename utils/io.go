package utils

import (
	"embed"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"red-cloud/mod/gologger"
	"strings"
)

//go:embed redc-templates
var local embed.FS
var dirs []string

// File copies a single file from src to dst
func File(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

// Dir copies a whole directory recursively
func Dir(src string, dst string) (err error) {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 计算目标路径 (例如 src/a -> dst/a)
		target := filepath.Join(dst, strings.TrimPrefix(path, src))
		info, _ := d.Info() // 忽略 info 获取错误，极大简化代码

		// 1. 目录处理 (包含根目录自身)
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		// 2. 软链接处理
		if info.Mode()&os.ModeSymlink != 0 {
			link, _ := os.Readlink(path)
			return os.Symlink(link, target)
		}

		// 3. 文件处理 (流式复制 + 权限保留)
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err = io.Copy(out, in); err != nil {
			return err
		}
		return os.Chmod(target, info.Mode()) // 关键：Terraform Provider 需要执行权限
	})
}

func GetFilesAndDirs(dirPth string) (files []string, dirs []string) {
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		gologger.Error().Msgf("获取目录失败: %s", err)
		return []string{}, []string{}
	}

	PthSep := string(os.PathSeparator)
	//suffix = strings.ToUpper(suffix) //忽略后缀匹配的大小写

	for _, fi := range dir {
		if fi.IsDir() { // 目录, 递归遍历
			dirs = append(dirs, dirPth+PthSep+fi.Name())
			GetFilesAndDirs(dirPth + PthSep + fi.Name())
		} else {
			// 过滤指定格式
			ok := strings.HasSuffix(fi.Name(), ".go")
			if ok {
				files = append(files, dirPth+PthSep+fi.Name())
			}
		}
	}

	return files, dirs
}

// ReleaseDir 释放文件夹
func ReleaseDir(path string) error {
	dirs, _ := local.ReadDir(path)
	for _, entry := range dirs {
		if entry.IsDir() {
			//_ = utils.Dir(path+"/"+entry.Name(), path+"/"+entry.Name())
			err := os.MkdirAll(path+"/"+entry.Name(), os.ModePerm)
			if err != nil {
				fmt.Println(err)
				return err
			}
			ReleaseDir(path + "/" + entry.Name())
		} else {
			//_ = utils.File(path+"/"+entry.Name(), path+"/"+entry.Name())
			out, _ := os.Create(path + "/" + entry.Name())
			in, _ := local.Open(path + "/" + entry.Name())
			_, err := io.Copy(out, in)
			if err != nil {
				fmt.Println(err)
				return err
			}
			in.Close()
		}
	}
	return nil
}

// ChechDirMain 递归
func ChechDirMain(dirPth string) []string {
	ChechDirSub(dirPth)
	return dirs
}

func ChechDirSub(dirPth string) {
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		os.Exit(3)
	}

	PthSep := string(os.PathSeparator)

	for _, fi := range dir {
		if fi.IsDir() { // 目录, 递归遍历
			dirs = append(dirs, dirPth+PthSep+fi.Name())
			ChechDirSub(dirPth + "/" + fi.Name())
		}
	}

}

func CheckFileName(path string, key string) bool {
	files, err := ioutil.ReadDir(path)
	if err == nil {
		for _, fileInfo := range files {
			if strings.Contains(fileInfo.Name(), key) {
				return true
			}
		}
	}
	return false
}
