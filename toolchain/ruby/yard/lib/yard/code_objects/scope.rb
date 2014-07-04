module YARD::CodeObjects
  module MultipleLocalScopes
    attr_reader :local_scopes

    def new_local_scope(name = "", parent = nil)
      name ||= ""
      @local_scopes_by_name ||= {}
      @local_scopes ||= []
      @local_scopes_by_name[name] ||= 0
      origname = name
      name += "_local_#{@local_scopes_by_name[name]}"
      ls = LocalScope.new(name, parent)
      @local_scopes << ls
      @local_scopes_by_name[origname] += 1
      ls
    end
  end

  class LocalScope
    def initialize(name, parent = nil)
      raise ArgumentError, "Invalid parent_scope: #{parent}" if parent && !parent.is_a?(LocalScope) && !parent.is_a?(NamespaceObject)
      @name = name
      @parent = parent
      @children = []
    end

    def root?; false end
    def has_tag?(_); false end
    def tag(name); end
    def tags(name = nil); end

    def path
      @path ||= if parent && !parent.root?
        [parent.path, name.to_s].join(sep)
      else
        name.to_s
      end
    end

    def sep; ">" end

    def name(prefix = false)
      prefix ? "#{sep}#{super}" : super
    end

    def parent_module
      if !parent
        nil
      elsif parent.is_a?(ModuleObject)
        parent
      else
        parent.parent_module
      end
    end

    attr_reader :name
    attr_reader :parent
    attr_reader :children

    def resolve(target)
      children.find do |c|
        c.name.to_s == target.to_s
      end || (parent.respond_to?(:resolve) && parent.resolve(target))
    end
  end
end
