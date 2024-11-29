package main

import (
"encoding/json"
"errors"
"fmt"
"log"
"os"
"path/filepath"
"sync"

"github.com/jcelliott/lumber"
)

const Version = "1.0.0"

type (
Logger interface {
Fatal(string, ...interface{})
Error(string, ...interface{})
Info(string, ...interface{})
Debug(string, ...interface{})
Warn(string, ...interface{})
Trace(string, ...interface{})
}

Driver struct {
mutex sync.Mutex
mutexes map[string]*sync.Mutex
dir string
log Logger
}
)

type Options struct {
Logger
}

func New(dir string, options *Options) (*Driver, error) {
dir = filepath.Clean(dir)

opts := Options{}
if options != nil {
opts = *options
}
if opts.Logger == nil {
opts.Logger = lumber.NewConsoleLogger(lumber.INFO)
}

driver := Driver{
dir: dir,
mutexes: make(map[string]*sync.Mutex),
log: opts.Logger,
}
if _, err := os.Stat(dir); err == nil {
opts.Logger.Info("Database directory already exists", dir)
return &driver, nil
}

opts.Logger.Debug("Creating database directory...", dir)
return &driver, os.MkdirAll(dir, 0755)
}

func (d *Driver) Write(collection string, resource string, v interface{}) error {
if collection == "" {
return errors.New("collection name cannot be empty")
}
if resource == "" {
return errors.New("resource name cannot be empty")
}
mutex := d.getOrCreateMutex(collection)
mutex.Lock()
defer mutex.Unlock()

dir := filepath.Join(d.dir, collection)
finalPath := filepath.Join(dir, resource+".json")
tempPath := finalPath + ".tmp"

if err := os.MkdirAll(dir, 0755); err != nil {
return err
}
b, err := json.MarshalIndent(v, "", "\t")
if err != nil {
return err
}

b = append(b, byte('\n'))
if err := os.WriteFile(tempPath, b, 0644); err != nil {
return err
}

return os.Rename(tempPath, finalPath)
}

func (d *Driver) ReadAll(collection string) (map[string]string, error) {
d.mutex.Lock()
defer d.mutex.Unlock()

dir := filepath.Join(d.dir, collection)
files, err := os.ReadDir(dir)
if err != nil {
return nil, err
}

records := make(map[string]string)
for _, file := range files {
if filepath.Ext(file.Name()) == ".json" {
data, err := os.ReadFile(filepath.Join(dir, file.Name()))
if err != nil {
return nil, err
}
records[file.Name()] = string(data)
}
}
return records, nil
}

func (d *Driver) Read(collection string, resource string, v interface{}) error {
if collection == "" {
return errors.New("collection name cannot be empty")
}
if resource == "" {
return errors.New("resource name cannot be empty")
}

mutex := d.getOrCreateMutex(collection)
mutex.Lock()
defer mutex.Unlock()

record := filepath.Join(d.dir, collection, resource+".json")
if _, err := os.Stat(record); os.IsNotExist(err) {
return errors.New("resource not found")
}

b, err := os.ReadFile(record)
if err != nil {
return err
}
return json.Unmarshal(b, &v)
}

func (d *Driver) Delete(collection string, resource string) error {
if collection == "" {
return errors.New("collection name cannot be empty")
}
if resource == "" {
return errors.New("resource name cannot be empty")
}

mutex := d.getOrCreateMutex(collection)
mutex.Lock()
defer mutex.Unlock()

record := filepath.Join(d.dir, collection, resource+".json")
if _, err := os.Stat(record); os.IsNotExist(err) {
return errors.New("resource not found")
}

return os.Remove(record)
}

func (d *Driver) getOrCreateMutex(collection string) *sync.Mutex {
d.mutex.Lock()
defer d.mutex.Unlock()
mutex, ok := d.mutexes[collection]

if !ok {
mutex = &sync.Mutex{}
d.mutexes[collection] = mutex
}
return mutex
}

type User struct {
Name string
Age json.Number
Address Address
}

type Address struct {
City string
State string
Country string
Pincode json.Number
}

func main() {
dir := "./DB"

db, err := New(dir, nil)
if err != nil {
log.Fatalf("Failed to initialize database: %v", err)
}

sampleEmployees := []User{
{"John Doe", "25", Address{"New York", "NY", "USA", "10001"}},
{"Jane Doe", "30", Address{"San Francisco", "CA", "USA", "94101"}},
{"John Smith", "35", Address{"Los Angeles", "CA", "USA", "90001"}},
{"Jane Smith", "40", Address{"Chicago", "IL", "USA", "60007"}},
}

for _, employee := range sampleEmployees {
if err := db.Write("users", employee.Name, employee); err != nil {
log.Printf("Failed to add employee: %s, error: %v", employee.Name, err)
} else {
log.Printf("Added: %s", employee.Name)
}
}

records, err := db.ReadAll("users")
if err != nil {
log.Fatalf("Failed to read all records: %v", err)
}

allUsers := []User{}

for _, v := range records {
employeeFound := User{}
if err := json.Unmarshal([]byte(v), &employeeFound); err != nil {
log.Fatalf("Failed to unmarshal data: %v", err)
} else {
allUsers = append(allUsers, employeeFound)
}
}

fmt.Println("All Users:", allUsers)
}