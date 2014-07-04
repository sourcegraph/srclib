require 'bundler'

module YARD
  module CLI
    class Bundle < Command
      def initialize
        @rebuild = false
        @gems = []
      end

      def description; "Builds YARD index for gems in bundle" end

      # Runs the commandline utility, parsing arguments and generating
      # YARD indexes for gems.
      #
      # @param [Array<String>] args the list of arguments
      # @return [void]
      def run(*args)
        require 'rubygems'
        optparse(*args)
        build_gems
      end

      private

      # Builds .yardoc files for all non-existing gems
      # @param [Array] gems
      def build_gems
        require 'rubygems'
        @gems.each do |spec|
          ver = "= #{spec.version}"
          dir = Registry.yardoc_file_for_gem(spec.name, ver)
          yfile = Registry.yardoc_file_for_gem(spec.name, ver, true)
          if @list
            puts "#{spec.name}\t#{yfile}"
          elsif dir && File.directory?(dir) && !@rebuild
            log.warn "#{spec.name} index already exists at '#{dir}'"
          else
            next unless yfile
            next unless File.directory?(spec.full_gem_path)
            Registry.clear
            Dir.chdir(spec.full_gem_path)
            log.warn "Building yardoc index for gem: #{spec.full_name} in #{yfile}"
            Yardoc.run('--no-stats', '-n', '-b', yfile)
          end
        end
      end

      def add_gems(gems)
        specs = Bundler.load.specs
        gems.each do |gem|
          s = specs[gem]
          if s.empty?
            log.warn "#{gem} could not be found in the Gemfile"
          end
          @gems += s
        end
      end

      # Parses options
      def optparse(*args)
        opts = OptionParser.new
        opts.banner = 'Usage: yard bundle [options] [gem_name]'
        opts.separator ""
        opts.separator "#{description}. If no gem_name is given,"
        opts.separator "all gem bundle dependencies are built."
        opts.separator ""
        opts.on('--list', 'list yardoc files') do
          @list = true
        end

        common_options(opts)
        parse_options(opts, args)
        add_gems(args)


        if !args.empty? && @gems.empty?
          log.error "No specified gems could be found for command"
        elsif @gems.empty?
          @gems += Bundler.load.specs.to_a if @gems.empty?
        end
      end
    end
  end
end
