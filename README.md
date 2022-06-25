# Sakuin (索引)
Sakuin is an http file indexer written in Go. 
It exposes your files from a given directory, simply and nicely.

## Building

```
go build -o build/sakuin
```

## Usage

```
Usage:
  sakuin [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  serve       Start the HTTP server

Flags:
      --config string   config file (default is $HOME/.sakuin.yaml)
  -h, --help            help for sakuin
  -t, --toggle          Help message for toggle
```

## Building Docker image

```
GOOS=linux go build -o build/sakuin
docker build .
```
