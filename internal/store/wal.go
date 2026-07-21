package store

import(
	"fmt"
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type WAL struct{
	path string
	file *os.File
	writer *bufio.Writer
}

func NewWAL(path string) (*WAL,error){
	file,err:=os.OpenFile(path,os.O_CREATE|os.O_WRONLY|os.O_APPEND,0o644)

	if err!=nil{
		return nil,err
	}

	return &WAL{
		path:path,
		file:file,
		writer:bufio.NewWriter(file),
	},nil
}

func (w *WAL) LogSet(key, value string) error {
    line := fmt.Sprintf("SET %s %s\n", key, value)
    _, err := w.writer.WriteString(line)
    return err
}

func (w* WAL) LogSetWithTTL(key,value string, expiry time.Time) error{
	expMs:=expiry.UnixMilli()
	line:=fmt.Sprintf("SET_TTL %s %s %d\n",key,value,expMs)
	_,err:=w.writer.WriteString(line)
	return err
}

func (w *WAL) LogDelete(key string) error {
	line:=fmt.Sprintf("DEL %s\n",key)
	_,err:=w.writer.WriteString(line)
	return err
}

func (w *WAL) Sync() error{
	if err:=w.writer.Flush(); err!=nil{
		return err
	}
	return w.file.Sync()
}

func (w *WAL) Close() error{
	if err:=w.writer.Flush();err!=nil{
		return err
	}
	return w.file.Close()
}

func ReplayWAL(walPath string, store *Store) error {
    file, err := os.Open(walPath)
    if os.IsNotExist(err) {
        return nil
    }
    if err != nil {
        return err
    }
    defer file.Close()

    now := time.Now()
    
    scanner := bufio.NewScanner(file)

    for scanner.Scan() {
        line := scanner.Text() 
        
        if line == "" {
            continue 
        }

        parts := strings.Fields(line)
        if len(parts) == 0 {
            continue
        }

        op := parts[0]
        
        switch op {
        case "SET":
            if len(parts) >= 3 {
                key := parts[1]
                value := strings.Join(parts[2:], " ")
                store.set(key, value, time.Time{})
            }

        case "SET_TTL":

            if len(parts) >= 4 {
                key := parts[1]
                // timestamp is the last field; everything between key and it is the value
                value := strings.Join(parts[2:len(parts)-1], " ")

                expMs, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
                if err != nil {
                    continue
                }
                expiry := time.UnixMilli(expMs)

                // already expired while the process was down: don't resurrect it
                if !now.Before(expiry) {
                    continue
                }

                store.set(key, value, expiry)
            }

        case "DEL":
            if len(parts) >= 2 {
                store.del(parts[1])
            }
        }
    }

    return scanner.Err()
}