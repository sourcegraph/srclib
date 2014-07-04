default: dep

dep:
	cd yard && ~/.rvm/bin/rvm 2.0.0-p353 do bundle
	~/.rvm/bin/rvm 2.0.0-p353 do gem install asciidoctor --no-rdoc --no-ri
	cd rubysonar && mvn clean package
