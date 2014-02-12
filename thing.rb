require 'bundler/setup'
require 'eventmachine'
require 'json'
require 'set'

$uid = rand(100000)

FD = 3
PIPE_READ = 4
PIPE_WRITE = 5

class Server
  def initialize(handler_klass, handler_options)
    @handler_klass, @handler_options = handler_klass, handler_options
    @connections = Set.new
    @onempty_callback = nil
    
    handler_options[:server] = self
  end
  
  def start
    @server = if ENV["LISTEN_FDS"] && ENV["LISTEN_FDS"].to_i == 1
      pipe = EM::BiPipe.new(PIPE_READ, PIPE_WRITE)
      @handler_options[:pipe] = pipe
      
      pipe.on_command { |c, args|
        p [:got_command, c, args]
        pipe.send_command("hello", "world")
      }
      
      # Accept on the passed file descriptor
      puts "Binding app to passed file descriptor"
      EM.attach_server(FD, @handler_klass, @handler_options)
    else
      puts "Starting new server on port 8000"
      EM.start_server('0.0.0.0', 8000, @handler_klass, @handler_options)
    end
  end
  
  def conn_add(c)
    @connections.add(c)
    report
  end
  
  def conn_rem(c)
    @connections.delete(c)
    report
    @onempty_callback.call if @onempty_callback
  end
  
  def stop_accepting(&onempty)
    report
    EM.stop_server(@server) if @server
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
  
  def report
    puts "Connections open: #{@connections.size}"
  end
  
  private
  
  def register_onempty(blk)
    @connections.empty? ? blk.call : @onempty_callback = blk
  end
end

class AppHandler < EM::Connection
  def initialize(options)
    @pipe = options[:pipe]
    @server = options[:server]
  end

  def post_init
    @server.conn_add(self)
    send_data("Hello there (#{$uid})")
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

class EM::BiPipe
  class PipeHandler < EM::Connection
    def initialize(pipe = nil)
      @pipe = pipe
    end

    def receive_data(data)
      parsed = JSON.parse(data)
      @pipe.command(*parsed)
    end
  end

  def initialize(r, w)
    @writer = EM.attach(w)
    @reader = EM.attach(r, PipeHandler, self)
  end

  def command(c, args)
    @on_command ? @on_command.call(c, args) : raise("Add on_command")
  end

  def on_command(&blk); @on_command = blk; end

  def send_command(command, args = {})
    @writer.send_data(JSON.generate([command, args]))
  end
end

EM.run do
  server = Server.new(AppHandler, {})
  server.start
  
  Signal.trap("HUP") do
    puts "HUP: Stop accepting"
    server.stop_accepting
  end
  
  try_graceful = true
  %w{INT TERM}.each do |sig|
    Signal.trap(sig) do
      if try_graceful
        puts "INT/TERM: Closing connections gracefully"
        server.close_gracefully { puts "Graceful exit"; EM.stop }
        try_graceful = false
      else
        puts "INT/TERM: Closing connections forcefully"
        server.close_forcefully { puts "Graceful exit"; EM.stop }
      end
    end
  end

end
