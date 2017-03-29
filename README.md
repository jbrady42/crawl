## Install

```
go get -u github.com/jbrady42/crawl/cmd/crawl
```

## Run

### Download
```
echo "https://google.com" > urls.txt

crawl download < urls.txt > crawl.data
```

### Extract Links
```
crawl extract < crawl.data > new_urls.txt
```

### All in One (Batch Mode)

```
crawl download < urls.txt | crawl extract > new_urls.txt

```