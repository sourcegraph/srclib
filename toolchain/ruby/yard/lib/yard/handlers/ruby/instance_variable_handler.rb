# Handles a instance variable (@@variable)
class YARD::Handlers::Ruby::InstanceVariableHandler < YARD::Handlers::Ruby::Base
  handles :assign

  process do
    if statement[0].type == :var_field && statement[0][0].type == :ivar
      name = statement[0][0][0]

      if self_binding == :class
        if owner.is_a?(MethodObject)
          object_klass = ClassVariableObject
          name.gsub!(/^@+/, '@@')
        else
          # we don't have a good way of naming an ivar in the class body, so just
          # ignore it. these are rare anyway.
          break
        end
      else
        object_klass = InstanceVariableObject
      end

      value = statement[1].source
      register object_klass.new(namespace, name) do |o|
        o.source = statement
        o.value = value
      end
    end
  end
end
