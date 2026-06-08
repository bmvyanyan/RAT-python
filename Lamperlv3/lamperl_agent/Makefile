all: clean
	@ echo "    * Building lamperl_agent plugin"
	@ mkdir dist
	@ cp config.json ax_config.axs ./dist/
	@ cp -r src_lamperl ./dist/
	@ go build -buildmode=plugin -ldflags="-s -w" -o ./dist/lamperl_agent.so pl_main.go pl_agent.go
	@ perl -c dist/src_lamperl/lamperl.pl
	@ echo "      done..."

clean:
	@ rm -rf dist
