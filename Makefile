clean:
	sudo rm -rf ./tmp/*.avi

run: clean
	go run main.go
