package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/yeka/zip"
)

type ChapterInfo struct {
	BookID   string     `json:"bookId"`
	Synckey  int        `json:"synckey"`
	Chapters []Chapters `json:"chapters"`
}
type Chapters struct {
	ChapterUID  int    `json:"chapterUid"`
	ChapterIdx  int    `json:"chapterIdx"`
	UpdateTime  int    `json:"updateTime"`
	Title       string `json:"title"`
	WordCount   int    `json:"wordCount"`
	Price       int    `json:"price"`
	IsMPChapter int    `json:"isMPChapter"`
	Paid        int    `json:"paid"`
}

func main() {
	vid, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("vid参数错误")
		return
	}
	dirPath := os.Args[2]

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if f.Name()[len(f.Name())-3:] == "res" {
			decryptFile(vid, dirPath+"/"+f.Name())
		}
	}

}
func getKeyAndIV(vid int) ([]byte, []byte) {

	remapArr := [10]byte{0x2d, 0x50, 0x56, 0xd7, 0x72, 0x53, 0xbf, 0x22, 0xfb, 0x20}
	vidLen := len(strconv.Itoa(vid))
	vidRemap := make([]byte, vidLen)

	for i := 0; i < vidLen; i++ {
		vidRemap[i] = remapArr[strconv.Itoa(vid)[i]-'0']
	}
	key := make([]byte, 36)
	for i := 0; i < 36; i++ {
		key[i] = vidRemap[i%vidLen]
	}
	iv := make([]byte, 16)
	for i := 0; i < 16; i++ {
		iv[i] = key[i+7]
	}
	key = key[0:16]
	return key, iv
}
func decryptFile(vid int, filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	readInt(f)
	chapterLen := readInt(f)
	chapterUid := make([]int, chapterLen)
	for i := 0; i < chapterLen; i++ {
		chapterUid[i] = readInt(f)
	}
	fmt.Println("章节Uid", chapterUid)
	readInt(f)
	encryptData := make([]byte, 16)
	f.Read(encryptData)
	key, iv := getKeyAndIV(vid)
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println(err)
		return
	}
	blockMode := cipher.NewCBCDecrypter(block, iv)
	decryptedData := make([]byte, 16)
	blockMode.CryptBlocks(decryptedData, encryptData)
	pwdStr := ""
	for i := 0; i < len(decryptedData); i++ {
		if decryptedData[i] < 32 || decryptedData[i] > 126 {
			continue
		}
		pwdStr += string(decryptedData[i])
	}
	f.Read(make([]byte, 8))
	zipFile := make([]byte, 0)
	for {
		b := make([]byte, 1024)
		n, err := f.Read(b)
		if err != nil {
			break
		}
		zipFile = append(zipFile, b[:n]...)
	}
	err = os.MkdirAll("./output", 0777)
	if err != nil {
		fmt.Println(err)
		return
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipFile), int64(len(zipFile)))
	if err != nil {
		fmt.Println(err)
		return
	}
	var chapterInfo ChapterInfo

	for i, f := range zipReader.File {
		if f.IsEncrypted() {
			f.SetPassword(pwdStr)
		}
		r, err := f.Open()
		if err != nil {
			log.Fatal(err)
		}
		if f.Name == "info.txt" {
			b, err := ioutil.ReadAll(r)
			if err != nil {
				fmt.Println(err)
				return
			}
			json.Unmarshal(b, &chapterInfo)

			continue
		}
		fileName := fmt.Sprintf("%d%s.txt", chapterInfo.Chapters[i-1].ChapterIdx-1, chapterInfo.Chapters[i-1].Title)
		_, err = os.Stat("./output/" + fileName)
		if err == nil {
			continue
		}
		file, err := os.Create("./output/" + fileName)
		if err != nil {
			fmt.Println(err)
			return
		}
		b, err := ioutil.ReadAll(r)
		if err != nil {
			fmt.Println(err)
			return
		}
		file.Write(b)
		file.Close()
	}
}
func readInt(f *os.File) int {
	b := make([]byte, 4)
	f.Read(b)
	return int(b[0])<<24 + int(b[1])<<16 + int(b[2])<<8 + int(b[3])
}
