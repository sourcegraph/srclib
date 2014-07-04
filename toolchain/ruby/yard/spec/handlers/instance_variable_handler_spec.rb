require File.dirname(__FILE__) + '/spec_helper'

describe "YARD::Handlers::Ruby::InstanceVariableHandler" do
  before(:all) { parse_file :instance_variable_handler_001, __FILE__ }

  it "should ignore instance variables inside class body" do
    Registry.at("A::B::@@inclass").should be_nil
    Registry.at("A::B::@inclass").should be_nil
  end

  it "should parse instance variables inside instance methods" do
    obj = Registry.at("A::B::@inmethod")
    obj.should_not be_nil
    obj.source.should == "@inmethod = \"hi\""
    obj.value.should == '"hi"'
  end

  it "should parse instance variables inside class methods as class variables" do
    obj = Registry.at("A::B::@@inclassmethod")
    obj.source.should == "@inclassmethod = \"hey\""
    obj.value.should == '"hey"'
  end
end
