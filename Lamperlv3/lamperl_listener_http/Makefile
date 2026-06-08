all: clean
	@ echo "    * Building lamperl_listener_http plugin"
	@ mkdir dist
	@ cp config.json ax_config.axs ./dist/
	@ go build -buildmode=plugin -ldflags="-s -w" -o ./dist/lamperl_http.so pl_main.go pl_listener.go pl_http.go
	@ echo "      done..."

clean:
	@ rm -rf dist
