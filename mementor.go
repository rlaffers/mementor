/**
 * Package main provides all the functionality behind the mementor tool.
 *
 * Mementor is a command line utility for displaying, adding and deleting mementos. Mementos are
 * messages describing things we need to be reminded of regularly.
 *
 *
 * @author Richard Laffers <rlaffers@gmail.com
 * @copyright Richard Laffers <rlaffers@gmail.com,
 * @package default
 * @version $Id$
 */
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"text/tabwriter"
	"time"
)

type Memento struct {
	Msg      string
	Time     int64
	Priority int8
}

const (
	VERSION = "0.1.2"
	HOME
)

var (
	dataFile string
	handles  map[string]*os.File
)

func init() {
	HOME := os.Getenv("HOME")
	if len(HOME) < 1 {
		panic("HOME environment variable is undefined")
	}
	// parse flags
	flag.StringVar(&dataFile, "f", HOME+"/.mementor/mementos.json", "Path to mementos storage file")
	flag.Parse()
	handles = make(map[string]*os.File)

	// create file if it does not exist
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		fmt.Printf("%s will be created\n", dataFile)
		_, err = createFile()
		if err != nil {
			panic("Failed to create data file: " + dataFile)
		}
	}

}

func main() {

	var err error

	defer func() {
		// close all open files
		for key, writer := range handles {
			//fmt.Printf("Closing %s handle\n", key)
			writer.Close()
			delete(handles, key)
		}
	}()

	args := flag.Args()

	var command string
	if len(args) > 0 {
		command = args[0]
	} else {
		command = "fetch"
	}
	switch command {
	case "add":
		err = add()
	case "fetch":
		fetch()
	case "rm", "del":
		err = rm()
	case "list", "ls":
		err = list()
	case "version":
		fmt.Println(VERSION)
	case "help":
		help()
	default:
		fmt.Printf("Action `%s` is invalid\n", command)
		help()
	}

	if err != nil {
		fmt.Println(err)
	}

	return
}

// print help screen
func help() {
	usage := `
Usage: mementor [OPTIONS...] ACTION [arguments...]

ACTIONS
	add		Add new memento.

	fetch	Display a random memento.

	rm		Remove a memento.

	help		Display this help.

	list		List all mementos.

	version		Display the current version.

OPTIONS
	-f			Path to the mementos storage file. Defaults to "$HOME/.mementor/mementos.json"
`
	fmt.Println(usage)
}

// list all mementos
func list() (err error) {
	mementos, err := readMementos()

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 5, 0, 1, ' ', 0)
	var formattedTime string
	for i, item := range mementos {
		formattedTime = time.Unix(item.Time, 0).Format("Jan 2 2006 15:04:05")
		fmt.Fprintf(w, "%d\t%s\t[%s]\n", i, item.Msg, formattedTime)
	}
	fmt.Fprintf(w, "--\n%d mementos shown.\n", len(mementos))
	w.Flush()
	return
}

// print a single random memento message
func fetch() (err error) {
	var n int
	mementos, err := readMementos()
	if err != nil {
		return err
	}
	if len(mementos) < 1 {
		return
	} else {
		rand.Seed(time.Now().Unix())
		n = rand.Intn(len(mementos))
	}
	fmt.Println(mementos[n].Msg)
	return
}

// add a new memento to the stack
func add() (err error) {
	var (
		memento Memento
		args    []string
	)
	args = flag.Args()
	if len(args) < 2 {
		return errors.New("Please specify the message.")
	}
	memento = Memento{
		Msg:      args[1],
		Time:     time.Now().Unix(),
		Priority: 1}
	mementos, err := readMementos()
	if err != nil {
		return err
	}
	mementos = append(mementos, memento)
	err = writeMementos(mementos)
	return err
}

// remove a memento from the stack
func rm() (err error) {
	var (
		mementos []Memento
		args     []string = flag.Args()
	)
	if len(args) < 2 {
		return errors.New("Please specify memento number.")
	}
	n, err := strconv.ParseInt(args[1], 10, 0)
	if err != nil || n < 0 {
		return errors.New("Invalid memento number: " + args[1])
	}

	// read all mementos
	mementos, err = readMementos()
	if err != nil {
		return err
	}
	// remove the Nth memento
	if n > int64(len(mementos)-1) {
		return fmt.Errorf("Memento %d does not exist.", n)
	}
	before := mementos[:n]
	after := mementos[n+1:]
	mementos = append(before, after...)
	writeMementos(mementos)
	return
}

// return parsed mementos from the passed file
func readMementos() (mementos []Memento, err error) {
	file, err := getFileReader()
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(file)
	err = dec.Decode(&mementos)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return mementos, nil
}

// write mementos into the file as a JSON string
func writeMementos(mementos []Memento) (err error) {
	var file *os.File
	// truncate the file
	file, err = createFile()
	if err != nil {
		return err
	}

	s, err := json.Marshal(mementos)
	if err != nil {
		return err
	}
	written, err := file.Write(s)
	fmt.Printf("%d bytes written\n", written)
	return
}

// create empty file or truncates an existing file
func createFile() (file *os.File, err error) {
	// create directory if necessary
	dir := filepath.Dir(dataFile)
	if _, err = os.Stat(dir); err != nil {
		fmt.Printf("Creating directory %s\n", dir)
		err = os.MkdirAll(dir, os.ModeDir|0700)
		if err != nil {
			return nil, fmt.Errorf("Failed to create directory for the data file at %s.\n%s", dir, err)
		}
	}
	file, err = os.Create(dataFile)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// get file handle for reading mementos
func getFileReader() (r *os.File, err error) {
	var ok bool
	if r, ok = handles["read"]; ok {
		return r, nil
	}
	r, err = os.Open(dataFile)
	if err == nil {
		handles["read"] = r
		return r, nil
	}
	return nil, err
}
