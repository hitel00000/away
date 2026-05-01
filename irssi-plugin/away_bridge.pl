use strict;
use warnings;

use Irssi;
use IO::Socket::UNIX;
use Time::HiRes qw(time);
use Fcntl qw(O_RDONLY O_NONBLOCK);

our $VERSION = '0.3';

our %IRSSI = (
  authors     => 'Away',
  name        => 'away_bridge',
  description => 'D-003a irssi bridge',
  license     => 'MIT',
);

#
# -------------------------
# paths
# -------------------------
#

my $EVENT_SOCKET = "/tmp/away/irc-companion.sock";
my $CMD_FIFO     = "/tmp/away/irc-companion.cmd";

my $cmd_fh;

# -------------------------
# outbound transport state
# -------------------------

our @outbound_queue = ();
our $MAX_QUEUE      = 100;
our $event_sock;

# D-003a: FIFO queue for opaque client_id correlation.
# Assumption: Irssi signals for own messages arrive in the same order
# as the commands were issued. This avoids text matching or heuristics.
our @pending_ids = ();

#
# -------------------------
# utility
# -------------------------
#

sub event_id {
  return "evt_" . int(time()*1000) . "_" . int(rand(100000));
}

sub iso_ts {
  my @t = gmtime();
  return sprintf(
      "%04d-%02d-%02dT%02d:%02d:%02dZ",
      $t[5]+1900,$t[4]+1,$t[3],
      $t[2],$t[1],$t[0]
  );
}

sub json_escape {
  my ($s)=@_;
  $s='' unless defined $s;

  $s =~ s/\\/\\\\/g;
  $s =~ s/"/\\"/g;
  $s =~ s/\n/\\n/g;
  $s =~ s/\r/\\r/g;
  $s =~ s/\t/\\t/g;

  return $s;
}

#
# -------------------------
# outbound transport
# -------------------------
#

sub flush_queue {

  return 1 unless @outbound_queue;

  # Try to connect if not connected
  unless ($event_sock) {
      $event_sock = IO::Socket::UNIX->new(
          Type    => SOCK_STREAM,
          Peer    => $EVENT_SOCKET,
          Timeout => 1,
      );
      if ($event_sock) {
          $event_sock->autoflush(1);
      }
  }

  unless ($event_sock) {
      return 1;
  }

  while (@outbound_queue) {
      my $line = $outbound_queue[0];

      # NOTE: We assume short NDJSON lines.
      # On UNIX sockets, writes smaller than PIPE_BUF (usually 4KB+) are typically atomic.
      # Partial writes are not explicitly handled here to keep it minimal.
      # SIGPIPE is generally handled/ignored by the irssi environment.
      if (print $event_sock $line . "\n") {
          shift @outbound_queue;
      } else {
          # Write failure: relay socket probably closed or broken
          $event_sock->close();
          undef $event_sock;
          last;
      }
  }

  return 1;
}

sub emit_json {
  my ($line)=@_;

  push @outbound_queue, $line;
  if (scalar @outbound_queue > $MAX_QUEUE) {
      shift @outbound_queue;
  }

  flush_queue();
}

#
# -------------------------
# event builders
# -------------------------
#

sub public_event_json {
  my ($server,$target,$nick,$text,$client_id)=@_;
  $client_id ||= "";

  return sprintf(
'{"type":"message.created","version":1,"id":"%s","timestamp":"%s","payload":{"network":"%s","buffer_id":"chan:%s","buffer_type":"channel","nick":"%s","text":"%s","highlight":false,"tags":[],"client_id":"%s"}}',
      event_id(),
      iso_ts(),
      json_escape($server->{tag}),
      json_escape($target),
      json_escape($nick),
      json_escape($text),
      json_escape($client_id),
  );
}

sub private_event_json {
  my ($server,$nick,$text,$client_id)=@_;
  $client_id ||= "";

  return sprintf(
'{"type":"dm.created","version":1,"id":"%s","timestamp":"%s","payload":{"network":"%s","peer":"%s","text":"%s","client_id":"%s"}}',
      event_id(),
      iso_ts(),
      json_escape($server->{tag}),
      json_escape($nick),
      json_escape($text),
      json_escape($client_id),
  );
}

sub sync_snapshot_json {
  my @buffers_json = ();
  
  for my $server (Irssi::servers()) {
      # Channels
      for my $chan ($server->channels()) {
          push @buffers_json, sprintf('{"id":"chan:%s","type":"channel","label":"%s"}', 
              json_escape($chan->{name}), json_escape($chan->{name}));
      }
      # Queries (DMs)
      for my $query ($server->queries()) {
          push @buffers_json, sprintf('{"id":"dm:%s","type":"dm","label":"%s"}', 
              json_escape($query->{name}), json_escape($query->{name}));
      }
  }

  return sprintf(
'{"type":"sync.snapshot","version":1,"id":"%s","timestamp":"%s","payload":{"buffers":[%s]}}',
      event_id(),
      iso_ts(),
      join(',', @buffers_json)
  );
}

sub emit_snapshot {
  emit_json(sync_snapshot_json());
}

#
# -------------------------
# irssi signal hooks
# -------------------------
#

# message public: SERVER_REC, char *msg, char *nick, char *address, char *target
sub on_public {
  my ($server, $text, $nick, $address, $target) = @_;
  emit_json(public_event_json($server, $target, $nick, $text, ""));
}

# message private: SERVER_REC, char *msg, char *nick, char *address, char *target
sub on_private {
  my ($server, $text, $nick, $address, $target) = @_;
  emit_json(private_event_json($server, $nick, $text, ""));
}

# message own_public: SERVER_REC, char *msg, char *target
sub on_own_public {
  my ($server, $text, $target) = @_;
  my $client_id = shift @pending_ids || "";
  emit_json(public_event_json($server, $target, $server->{nick}, $text, $client_id));
}

# message own_private: SERVER_REC, char *msg, char *target, char *orig_target
sub on_own_private {
  my ($server, $text, $target, $orig_target) = @_;
  my $client_id = shift @pending_ids || "";
  emit_json(private_event_json($server, $target, $text, $client_id));
}

#
# -------------------------
# inbound command fifo
# -------------------------
#

sub init_command_fifo {

  unless (-p $CMD_FIFO) {
      system("mkfifo",$CMD_FIFO);
  }

  sysopen(
      $cmd_fh,
      $CMD_FIFO,
      O_RDONLY | O_NONBLOCK
  ) or do {
      Irssi::print("away_bridge: fifo open failed");
      return;
  };

  Irssi::print("away_bridge command fifo ready");
}

# NOTE: We use regex for parsing instead of JSON::PP because some irssi
# installations run on very old Perl versions where JSON::PP is not available.
# This is an intentional limitation to maximize compatibility.
sub parse_send_message {

  my ($line)=@_;

  return unless $line =~ /"action":"send_message"/;

  my $client_id = '';
  if ($line =~ /"client_id":"([^"]+)"/) {
      $client_id = $1;
  }

  return unless $line =~ /"target":"([^"]+)"/;
  my $target=$1;

  return unless $line =~ /"text":"([^"]*)"/;
  my $text=$1;

  $text =~ s/\\"/"/g;
  $text =~ s/\\\\/\\/g;

  return ($target,$text,$client_id);
}

sub parse_mark_read {
  my ($line)=@_;
  return unless $line =~ /"action":"mark_read"/;
  return unless $line =~ /"target":"([^"]+)"/;
  return $1;
}

sub poll_commands {

  return 1 unless $cmd_fh;

  my $buf='';
  my $n = sysread($cmd_fh, $buf, 4096);

  return 1 unless defined $n && $n > 0;

  for my $line (split /\n/, $buf) {
  
      my $target_read = parse_mark_read($line);
      if ($target_read) {
          Irssi::print("away_bridge: mark_read $target_read (wire ack)");
          next;
      }

      my ($target,$text,$client_id)=parse_send_message($line);
      next unless $target;
  
      my @servers = Irssi::servers();
      next unless @servers;
  
      if ($client_id) {
          push @pending_ids, $client_id;
      }

      $servers[0]->command("msg $target $text");
  }

  return 1;
}

#
# -------------------------
# init
# -------------------------
#

Irssi::signal_add('message public', 'on_public');
Irssi::signal_add('message private', 'on_private');
Irssi::signal_add('message own_public', 'on_own_public');
Irssi::signal_add('message own_private', 'on_own_private');
Irssi::signal_add('channel joined', 'emit_snapshot');
Irssi::signal_add('channel parted', 'emit_snapshot');

init_command_fifo();
emit_snapshot();

Irssi::timeout_add(250, 'poll_commands', undef);
Irssi::timeout_add(5000, 'flush_queue', undef);

Irssi::print("away_bridge loaded (D-003a plumbing)");