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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/xeonx/timeago"
)

// Memento represents a record in mementor.
type Memento struct {
	Id       int
	Msg      string
	Time     int64
	Priority int8
}

const (
	version = "0.1.3"
)

type print struct {
}

func (p *print) info(msg string, args ...interface{}) {
	fmt.Printf("\x1b[36;1m"+msg+"\n\x1b[0m", args...)
}

func (p *print) error(msg string, args ...interface{}) {
	fmt.Printf("\x1b[31;1m"+msg+"\n\x1b[0m", args...)
}

func (p *print) underscore(msg string, args ...interface{}) {
	fmt.Printf("\x1b[4;1m"+msg+"\n\x1b[0m", args...)
}

var (
	dataFile *string
	debug    = flag.Bool("debug", false, "Turn debugging on.")
	logger   *logrus.Logger
	pr       = print{}
)

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		panic("HOME variable is not set")
	}

	dataFile = flag.String("f", home+"/.mementor/mementos.json", "Path to the mementos storage file.")
	// parse flags
	flag.Parse()
	logger = logrus.New()
	if *debug {
		logger.Level = logrus.DebugLevel
	} else {
		logger.Level = logrus.InfoLevel
	}
	formatter := new(logrus.TextFormatter)
	//formatter.FullTimestamp = true
	//formatter.TimestampFormat = "2006-01-02 15:04:05.000"
	logger.Formatter = formatter

	// create the mementos file if it does not exist
	// TODO presunut do createorappend
	if _, err := os.Stat(*dataFile); err != nil {
		if os.IsNotExist(err) {
			_, err = createFile()
			if err != nil {
				panic("Failed to create data file: " + *dataFile)
			}
			pr.info("%s was be created", *dataFile)

		} else {
			panic(err)
		}
	}

}

func main() {
	args := flag.Args()

	var command string
	if len(args) > 0 {
		command = args[0]
	} else {
		command = "fetch"
	}
	var err error
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
		fmt.Println(version)
	case "help":
		help()
	default:
		pr.error("Action `%s` is invalid", command)
		help()
	}
	if err != nil {
		pr.error(err.Error())
	}

	return
}

// print help screen
func help() {
	usage := `
Usage: mementor [OPTIONS...] ACTION [arguments...]

ACTIONS
	add			Add new memento.
	fetch		Display a random memento.
	rm			Remove a memento.
	help		Display this help.
	list		List all mementos.
	version		Display the current version.

OPTIONS
`
	fmt.Print(usage)
	flag.PrintDefaults()
}

// list all mementos
func list() error {
	mementos, err := readMementos()
	if err != nil {
		return err
	}

	pr.underscore(" ID   Age          Description")
	cfg := timeago.NoMax(timeago.English)
	cfg.PastSuffix = ""
	for _, m := range mementos {
		t := time.Unix(m.Time, 0)
		fmt.Printf("%3d   %10s   %s\n", m.Id, cfg.Format(t), m.Msg)
	}

	pr.info("\n%d mementos total.\n", len(mementos))
	return nil
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

// TODO modify

// add a new memento to the stack
func add() (err error) {
	var args []string = flag.Args()
	if len(args) < 2 {
		return errors.New("Please specify the message.")
	}
	// concat the remaining arguments as a message string
	msg := strings.Join(args[1:], " ")
	mementos, err := readMementos()
	var lastId int
	if len(mementos) < 1 {
		lastId = 0
	} else {
		lastMemento := mementos[len(mementos)-1]
		lastId = lastMemento.Id
	}
	m := Memento{
		Id:       lastId + 1,
		Msg:      msg,
		Time:     time.Now().Unix(),
		Priority: 1,
	}
	logger.Debugf("Writing %+v", m)
	if err != nil {
		return err
	}
	mementos = append(mementos, &m)
	err = writeMementos(mementos)
	return err
}

// remove a memento from the stack
func rm() error {
	var (
		args []string = flag.Args()
	)
	if len(args) < 2 {
		return errors.New("Please specify a memento ID.")
	}
	id64, err := strconv.ParseInt(args[1], 10, 0)
	if err != nil || id64 < 0 {
		return fmt.Errorf("Invalid memento Id: %v", args[1])
	}
	id := int(id64)

	// read all mementos
	mementos, err := readMementos()
	if err != nil {
		return err
	}
	// do a binary search for the memento. It should be sorted in ascending order
	count := len(mementos)
	n := sort.Search(count, func(i int) bool {
		return mementos[i].Id >= id
	})
	if n < count && mementos[n].Id == id {
		pr.info("found memento at %d", n)
	} else {
		// not found
		return fmt.Errorf("Memento %d does not exist", id)
	}

	before := mementos[:n]
	after := mementos[n+1:]
	mementos = append(before, after...)
	writeMementos(mementos)
	return nil
}

// return parsed mementos from the passed file
//func readMementos() ([]*Memento, error) {
func readMementos() ([]*Memento, error) {
	logger.Debugf("opening %s", *dataFile)
	r, err := os.Open(*dataFile)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(r)
	var mementos []*Memento
	if err := dec.Decode(&mementos); err != nil && err != io.EOF {
		return nil, err
	}
	return mementos, nil
}

// write mementos into the file as a JSON string
func writeMementos(mementos []*Memento) (err error) {
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
	dir := filepath.Dir(*dataFile)
	if _, err = os.Stat(dir); err != nil {
		fmt.Printf("Creating directory %s\n", dir)
		err = os.MkdirAll(dir, os.ModeDir|0700)
		if err != nil {
			return nil, fmt.Errorf("Failed to create directory for the data file at %s.\n%s", dir, err)
		}
	}
	file, err = os.Create(*dataFile)
	if err != nil {
		return nil, err
	}
	return file, nil
}
