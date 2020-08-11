## Mass RegExp Scanner - mres
A fast and configurable regular expression scanner to mass scan folder and file content - powered by Goroutines.

The project is work in progress...

### Usage
`Warning:` It works but not complete...

#### Steps

1. `git clone https://github.com/movna/mres`
2. `cd mres/cmd/mres`
3. `go build`
4. `./mres -path <folder_path> -regex <exp> -workers <no_of_workers>`

### Inspiration
I am learning Golang and thought building something like this is best use and test of what I am learning - specially on Goroutines.

### TODO
- [x] Basic scanning - outputs filepath to stdout if there is a match
- [x] Cancellation support - graceful exit
- [x] Support callbacks on results and errors
- [ ] Add line number and matched content to results
- [ ] Add config file support
- [ ] Write results to output file
- [ ] Add match file extensions
- [ ] Add file extension filters
- [ ] Async writes to log and results file
- [ ] Get someone who is experienced in Golang to review code :D
- [ ] Tests
- [ ] Make it go gettable
- [ ] As a module to be used by others
- [ ] Documentation
- [ ] Signature library for common use cases
- [ ] Optimize for big files
