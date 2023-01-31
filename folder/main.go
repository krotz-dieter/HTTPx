package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

func CopyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func PrepareFile(filelocation string, dirout string) error {
	// parse file and move to target folder
	if len(filelocation) > 0 {
		f, err := os.Open(filelocation)
		if err != nil {
			log.Printf("Error opening file: %s\n", filelocation)
			return err
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			log.Println("err reading", err)
			return err
		}

		var ds *dicom.Dataset
		data, err := dicom.Parse(f, info.Size(), nil)
		if err != nil {
			log.Fatalf("error parsing data: %v", err)
			return err
		}
		ds = &data

		// In non-streaming frame mode, we need to find all PixelData elements and generate images.
		log.Println(" File:: ", filelocation)
		var studyinstanceuid, seriesinstanceuid, sopinstanceuid string
		for _, elem := range ds.Elements {
			if elem.Tag == tag.StudyInstanceUID {
				studyinstanceuid = strings.Trim(elem.Value.String(), "[]")
				log.Println("   StudyInstanceUID: ", studyinstanceuid)
			} else if elem.Tag == tag.SeriesInstanceUID {
				seriesinstanceuid = strings.Trim(elem.Value.String(), "[]")
				log.Println("   SeriesInstanceUID: ", seriesinstanceuid)
			} else if elem.Tag == tag.SOPInstanceUID {
				sopinstanceuid = strings.Trim(elem.Value.String(), "[]")
				log.Println("   SOPInstanceUID: ", sopinstanceuid)
			}
		}
		// create directory
		if len(studyinstanceuid) > 0 && len(seriesinstanceuid) > 0 && len(sopinstanceuid) > 0 {
			// create target directory
			directoryname := filepath.Join(dirout, studyinstanceuid)
			_ = os.Mkdir(directoryname, os.ModePerm)
			directoryname = filepath.Join(directoryname, seriesinstanceuid)
			_ = os.Mkdir(directoryname, os.ModePerm)

			// copy file to target and rename
			filetarget := filepath.Join(directoryname, sopinstanceuid+".dcm")
			nBytes, err := CopyFile(filelocation, filetarget)
			if err != nil {
				log.Println("  Copy failed from:", filelocation, " to:", filetarget)
				return err
			}
			log.Println(" Result File: ", filetarget, " now available with size:", nBytes)
		} else {
			log.Println("  Ignoring file with wrong DICOM tags: ", filelocation)
		}
	}
	return nil
}

func PrepareDirectory(dirin string, dirout string) error {
	// build path
	log.Println("Processing directory: ", dirin)
	s := time.Now()

	// walk through all files
	errWalk := filepath.Walk(dirin, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("No files in directory: ", path, err)
			return err
		}

		// process every file
		if !info.IsDir() {
			sFile := time.Now()
			err := PrepareFile(path, dirout)
			if err != nil {
				log.Println("Error processing file: ", path, err)
			}
			log.Println("Reading file: ", path, " in: ", time.Since(sFile))
		}
		return nil
	})
	if errWalk != nil {
		log.Println("No files in directory: ", dirin)
		return errWalk
	}
	log.Println("Time total taken: ", time.Since(s))
	return nil
}

func main() {
	// additional parameters
	dirin := flag.String("dirin", "", "directory to be used as input")
	dirout := flag.String("dirout", "", "directory to be used as output")
	flag.Parse()
	if *dirin == "" || *dirout == "" {
		fmt.Println("Usage: httpx-folder.exe -dirin X -dirout Y")
		fmt.Println("flags:")
		flag.PrintDefaults()
		fmt.Println("Examples:")
		fmt.Println("Prepare folder: httpx-folder.exe -dirin d:\\in -dirout d:\\out")
		return
	}

	// check input directory
	_, err1 := os.Stat(*dirin)
	if err1 != nil {
		if os.IsNotExist(err1) {
			// File or directory does not exist
			panic("Directory " + *dirin + " does not exist. Please provide an existing directory !")
		}
	}

	// check input directory
	_, err2 := os.Stat(*dirout)
	if err2 != nil {
		if os.IsNotExist(err2) {
			// File or directory does not exist
			panic("Directory " + *dirout + " does not exist. Please provide an existing directory !")
		}
	}

	// check input directory
	err3 := PrepareDirectory(*dirin, *dirout)
	if err3 != nil {
		log.Println("Error in processing directory:", err3)
	}

}
