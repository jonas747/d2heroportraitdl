package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
)

type FeedResponse struct {
	Herodata map[string]interface{}
}

type DlResult struct {
	Img  image.Image
	Hero string
	Err  error
}

func main() {
	fmt.Println("Downloading listing...")
	result, err := getAllHeroPortraits()
	if err != nil {
		panic(err)
	}
	fmt.Println("Finnished downloading, saving...")
	err = saveImages(result)
	if err != nil {
		panic(err)
	}
	fmt.Println("Finnished!")
}

func saveImages(images []DlResult) error {
	exist, err := exists("portraits")
	if err != nil {
		return err
	}
	if !exist {
		err := os.Mkdir("portraits", os.ModeDir)
		if err != nil {
			return err
		}
	}
	err = os.Chdir("portraits")
	if err != nil {
		return err
	}

	for _, v := range images {
		if v.Img == nil {
			fmt.Println(v.Hero + "'s image is nil, skipping")
			continue
		}

		file, err := os.Create(v.Hero + ".png")
		if err != nil {
			return err
		}
		defer file.Close()

		err = png.Encode(file, v.Img)
		if err != nil {
			fmt.Println("Failed encoding " + v.Hero + ", Skipping")
			continue
		}
	}
	return nil
}

func downloadImage(res chan DlResult, hero string) {
	location := "http://media.steampowered.com/apps/dota2/images/heroes/" + hero + "_sb.png"
	resp, err := http.Get(location)
	if err != nil {
		res <- DlResult{nil, hero, err}
		return
	}
	defer resp.Body.Close()
	img, err := png.Decode(resp.Body)
	if err != nil {
		res <- DlResult{nil, hero, err}
		return
	}
	res <- DlResult{img, hero, nil}
}

func downloadImages(res chan DlResult, heroes []string) {
	for _, v := range heroes {
		downloadImage(res, v)
	}
}

func getAllHeroPortraits() ([]DlResult, error) {
	resp, err := http.Get("http://www.dota2.com/jsfeed/heropediadata?feeds=herodata")
	if err != nil {
		return make([]DlResult, 0), err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return make([]DlResult, 0), err
	}

	var heroData FeedResponse
	err = json.Unmarshal(body, &heroData)
	if err != nil {
		return make([]DlResult, 0), err
	}

	heroNameList := make([]string, 0)

	//Split the downloads between the number of cores you have
	for k, _ := range heroData.Herodata {
		heroNameList = append(heroNameList, k)
	}
	fmt.Printf("Downloading %d Hero portraits\n", len(heroNameList))

	resultChan := make(chan DlResult)
	numSplit := len(heroNameList) / runtime.NumCPU()
	last := 0
	for i := 0; i < runtime.NumCPU(); i++ {
		if i == runtime.NumCPU()-1 {
			//last cycle
			go downloadImages(resultChan, heroNameList[last:])
			break
		}
		now := numSplit * (i + 1)
		go downloadImages(resultChan, heroNameList[last:now])
		last = now
	}

	list := make([]DlResult, len(heroNameList)+1)
	for i := 0; i < len(heroNameList); i++ {
		result := <-resultChan
		if result.Err != nil {
			fmt.Println(result.Hero + ":" + result.Err.Error())
		}
		list[i] = result
		fmt.Printf("Finnished downloading hero prortrait of %s (%d/%d)\n", result.Hero, i+1, len(heroNameList))
	}

	return list, nil
}

// exists returns whether the given file or directory exists or not
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
