require File.dirname(__FILE__) + '/../spec_helper'

describe YARD::CLI::Condense do
  before do
    @condense = YARD::CLI::Condense.new
    @condense.use_cache = true
    YARD.stub!(:parse)
    Registry.stub!(:load!)
  end

  describe 'Running' do
    before do
      @condense = CLI::Condense.new
    end
  end

end
