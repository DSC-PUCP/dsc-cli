BIN := dsc
INSTALL_DIR := /usr/local/bin

.PHONY: build install uninstall

build:
	go build -o $(BIN) .

install: build
	cp $(BIN) $(INSTALL_DIR)/$(BIN)

uninstall:
	rm -f $(INSTALL_DIR)/$(BIN)
