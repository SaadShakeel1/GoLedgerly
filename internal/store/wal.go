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

func (w* WAL) LogSetWithTTL(key,value string, ttl time.Duration) error{
	ttlMs:=int64(ttl.Milliseconds())
	line:=fmt.Sprintf("SET_TTL %s %s %d\n",key,value,ttlMs)
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