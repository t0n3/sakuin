# Sakuin (索引)
Sakuin is an http file indexer written in Go. 
It to expose your files from a given directory, simply and nicely.

## Building

```
go build -o bin/sakuin
```

## Usage

```
Usage of sakuin:
  -dir string
        Path to data dir you want to expose (default ".")
  -port int
        Port binded by Sakuin (default 3000)
```

## Building Docker image

```
GOOS=linux go build -o bin/sakuin
docker build .
```
