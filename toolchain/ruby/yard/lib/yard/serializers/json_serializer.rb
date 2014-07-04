require 'json'
require 'pathname'

module YARD
  module Serializers
    class JSONSerializer < Base
      # only emit symbols defined in these files
      def initialize(files)
        @files = files.map do |f|
          f = Pathname.new(f)
          if f.directory?
            fs = []
            f.find { |p|
              fs << p if p.file?
            }
            fs
          else
            [f]
          end
        end.flatten.map(&:to_s)
      end

      def serialize(data)
        data = {
          :objects => data[:objects].select { |o| output_object?(o) }.map { |o| prepare_object(o) },
        }
        print(JSON.fast_generate(data))
      end

      def after_serialize
        print("\n")
      end

      def output_object?(object)
        object.parent_module && object.ast_node && @files.include?(object.ast_node.file)
      end

      def prepare_object(object)
        o = {
          :name => object.name,
          :path => object.path,
          :module => object.parent_module,
          :kind => object.type,
          :file => object.file,
          :exported => !object.name.to_s.start_with?('@'),
        }

        if object.ast_node.respond_to?(:source_range) && object.ast_node.source_range
          o[:defStart] = object.ast_node.source_range.first
          o[:defEnd] = object.ast_node.source_range.last + 1
        end

        if !object.docstring.empty?
          o[:docstring] = begin
                            object.format(:format => :html, :markup => :rdoc, :template => :sourcegraph)
                          rescue
                            "<!-- doc error -->"
                          end
        end

        case object.type
        when :method
          o[:signature] = object.signature.sub('def ', '') if object.signature
        end
        o
      end
    end
  end
end
