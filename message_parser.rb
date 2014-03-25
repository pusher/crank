class MessageParser
  def initialize
    @msgs = []

    @input = []
    @input_size = 0
  end

  def feed_data(data)
    @input.push(data)
    @input_size += data.bytesize

    while true
      message = parse_message
      if message
        @msgs.push(message)
      else
        break
      end
    end
  end

  def pull_message
    @msgs.shift
  end

  private

  class InsufficientInput < StandardError; end

  def parse_message
    begin
      return read(4) { |dat|
        length = dat.unpack("N")[0]
        read(length) { |msg|
          msg
        }
      }
    rescue InsufficientInput
      nil
    end
  end

  def read(n, &blk)
    if @input_size < n
      raise InsufficientInput
    else
      pieces = []
      left = n
      while left > 0
        d = @input.shift
        if d.bytesize > left
          pieces.push(d[0...left])
          @input.unshift(d[left..-1])
          @input_size -= left
          left = 0
        else
          pieces.push(d)
          left -= d.bytesize
          @input_size -= d.bytesize
        end
      end

      data = pieces.join('')
      begin
        blk.call(data)
      rescue InsufficientInput
        @input.unshift(data)
        @input_size += data.bytesize
        raise
      end
    end
  end
end
