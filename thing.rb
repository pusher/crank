require 'bundler/setup'
require 'eventmachine'
require 'json'
require 'set'
require './message_parser'

class BiPipe
  class PipeHandler < EM::Connection
    def initialize(pipe = nil)
      @pipe = pipe
      @message_parser = MessageParser.new
    end

    def receive_data(data)
      @message_parser.feed_data(data)
      while msg = @message_parser.pull_message
        @pipe.command(*JSON.parse(msg))
      end
    end
  end

  def initialize(r, w)
    @writer = EM.attach(IO.for_fd(w))
    @reader = EM.attach(IO.for_fd(r), PipeHandler, self)
  end

  def command(c, args)
    @on_command ? @on_command.call(c, args) : raise("Add on_command")
  end

  def on_command(&blk); @on_command = blk; end

  def send_command(command, args = {})
    data = JSON.generate([command, args])
    @writer.send_data("#{[data.bytesize].pack('N')}#{data}")
  end
end

class CrankedServer
  FD = 3
  PIPE_READ = 4
  PIPE_WRITE = 5

  attr_reader :under_crank

  # Delegate must respond to
  # * start_accepting(fd)
  # * start_server(port)
  # * stop_accepting(&onempty)
  # * close_gracefully(&onempty)
  # * close_forcefully(&onempty)
  def initialize(server_delegate, port)
    @server = server_delegate
    @port = port
    @accepting = false
    @stop_gracefully = true

    @under_crank = ENV["LISTEN_FDS"] && ENV["LISTEN_FDS"].to_i == 1
  end

  def run
    if @under_crank
      @pipe = BiPipe.new(PIPE_READ, PIPE_WRITE)
      @pipe.on_command(&method(:pipe_command))

      @pipe.send_command("STARTED")
    else
      # Fallback to starting accepting immediately in the absence of crank
      start_accepting()
    end
  end

  def start_accepting
    return if @accepting

    if @under_crank
      @server.start_accepting(FD)
    else
      @server.start_server(@port)
    end
    @accepting = true
  end

  # Simplified wrapper for close_gracefully & close_forcefully
  # Calls
  def stop(&onempty)
    if @stop_gracefully
      puts "RUBY: INT/TERM: Closing connections gracefully (#{@server.report})"
      @server.close_gracefully(&onempty)
      @stop_gracefully = false
    else
      puts "RUBY: INT/TERM: Closing connections forcefully (#{@server.report})"
      @server.close_forcefully(&onempty)
    end
  end

  def stop_accepting(&onempty)
    return unless @accepting

    @server.stop_accepting(&onempty)
    @accepting = false
    register_onempty(onempty) if onempty
  end

  private

  def pipe_command(command, args)
    puts "RUBY: Received pipe command #{command}, args: #{args}"
    case command
    when "START_ACCEPTING"
      start_accepting
    else
      puts "RUBY: Unknown pipe command #{command}"
    end
  end
end


class Server
  def initialize(handler_klass, handler_options)
    @handler_klass, @handler_options = handler_klass, handler_options
    @connections = Set.new
    @onempty_callback = nil

    handler_options[:server] = self
  end

  def conn_add(c)
    @connections.add(c)
  end

  def conn_rem(c)
    @connections.delete(c)
    @onempty_callback.call if @onempty_callback
  end

  def report
    "Connections open: #{@connections.size}"
  end

  def start_accepting(fd)
    puts "RUBY: Binding app to passed file descriptor"
    @server = EM.attach_server(fd, @handler_klass, @handler_options)
  end

  def start_server(port)
    puts "RUBY: Starting new server on port #{port}"
    @server = EM.start_server('0.0.0.0', port, @handler_klass, @handler_options)
  end

  def stop_accepting(&onempty)
    return unless @server

    EM.stop_server(@server)
    @server = nil
    register_onempty(onempty) if onempty
  end

  def close_gracefully(&onempty)
    @connections.each { |c| c.close_gracefully }
    register_onempty(onempty) if onempty
  end

  def close_forcefully(&onempty)
    @connections.each { |c| c.close_forcefully }
    register_onempty(onempty) if onempty
  end

  private

  def register_onempty(blk)
    @connections.empty? ? blk.call : @onempty_callback = blk
  end
end


class AppHandler < EM::Connection
  def initialize(options)
    @server = options[:server]
  end

  def post_init
    @server.conn_add(self)
    send_data("Hello there (#{Process.pid})")
    # @timer = EM::Timer.new(0.2) do
    #   close_connection
    # end
  end

  def unbind
    # @timer.cancel if @timer
    @server.conn_rem(self)
  end

  def close_gracefully
    send_data("close handshake")
  end

  def close_forcefully
    close_connection
  end
end

server = Server.new(AppHandler, {})
cranked_server = CrankedServer.new(server, 8000)

EM.run do
  cranked_server.run

  Signal.trap("HUP") do
    puts "RUBY: HUP: Stop accepting (#{server.report})"
    cranked_server.stop_accepting
  end

  %w{INT TERM}.each do |sig|
    Signal.trap(sig) do
      cranked_server.stop { puts "RUBY: Graceful exit"; EM.stop }
    end
  end
end
