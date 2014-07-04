module YARD::CodeObjects
  # Represents a instance variable inside a namespace. The path is expressed
  # in the form "A::B#@instancevariable"
  class InstanceVariableObject < Base
    # @return [String] the instance variable's value
    attr_accessor :value
  end
end
