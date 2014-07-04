require File.dirname(__FILE__) + '/spec_helper'

describe "YARD::Handlers::Ruby::#{LEGACY_PARSER ? "Legacy::" : ""}ClassVariableHandler" do
  before(:all) { parse_file :class_variable_handler_001, __FILE__ }

  it "should parse class variables inside classes" do
    obj = Registry.at("A::B::@@inclass")
    obj.source.should == "@@inclass = \"hello\""
    obj.value.should == '"hello"'
  end

  it "should parse class variables inside methods" do
    obj = Registry.at("A::B::@@inmethod")
    obj.source.should == "@@inmethod = \"hi\""
    obj.value.should == '"hi"'
  end
end
