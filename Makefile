all: 
	@go build -o rp
clean:
	rm -f rp
install: all
	cp rp /usr/local/bin	
uninstall: 
	rm -f /usr/local/bin/rp
