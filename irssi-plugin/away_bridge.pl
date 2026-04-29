use strict;
use warnings;

use Irssi;
use IO::Socket::UNIX;
use Time::HiRes qw(time);
use Fcntl qw(O_RDONLY O_NONBLOCK);

our $VERSION = '0.2';

our %IRSSI = (
  authors     => 'Away',
  name        => 'away_bridge',
  description => 'C-001/C-002 irssi bridge',
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

  my ($server,$target,$nick,$text)=@_;

  return sprintf(
'{"type":"message.created","version":1,"id":"%s","timestamp":"%s","payload":{"network":"%s","buffer_id":"chan:%s","buffer_type":"channel","nick":"%s","text":"%s","highlight":false,"tags":[]}}',
      event_id(),
      iso_ts(),
      json_escape($server->{tag}),
      json_escape($target),
      json_escape($nick),
      json_escape($text),
  );
}

sub private_event_json {

  my ($server,$nick,$text)=@_;

  return sprintf(
'{"type":"dm.created","version":1,"id":"%s","timestamp":"%s","payload":{"network":"%s","peer":"%s","text":"%s"}}',
      event_id(),
      iso_ts(),
      json_escape($server->{tag}),
      json_escape($nick),
      json_escape($text),
  );
}

#
# -------------------------
# irssi signal hooks
# -------------------------
#

sub on_public_message {

  my ($server,$text,$nick,$address,$target)=@_;

  emit_json(
      public_event_json(
          $server,
          $target,
          $nick,
          $text
      )
  );
}

sub on_private_message {

  my ($server,$text,$nick,$address)=@_;

  emit_json(
      private_event_json(
          $server,
          $nick,
          $text
      )
  );
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
sub parse_send_message {

  my ($line)=@_;

  return unless $line =~ /"action":"send_message"/;

  return unless $line =~ /"target":"([^"]+)"/;
  my $target=$1;

  return unless $line =~ /"text":"([^"]*)"/;
  my $text=$1;

  $text =~ s/\\"/"/g;
  $text =~ s/\\\\/\\/g;

  return ($target,$text);
}

sub poll_commands {

  return 1 unless $cmd_fh;

  my $buf='';

  my $n = sysread(
      $cmd_fh,
      $buf,
      4096
  );

  return 1 unless defined $n;
  return 1 if $n <= 0;

  for my $line (split /\n/, $buf) {
  
      my ($target,$text)=parse_send_message($line);
      next unless $target;
  
      my @servers = Irssi::servers();
      next unless @servers;
  
      $servers[0]->command(
          "msg $target $text"
      );
  }

  return 1;
}

#
# -------------------------
# init
# -------------------------
#

Irssi::signal_remove(
  'message public',
  'on_public_message'
);

Irssi::signal_remove(
  'message private',
  'on_private_message'
);

Irssi::signal_add(
  'message public',
  'on_public_message'
);

Irssi::signal_add(
  'message private',
  'on_private_message'
);

init_command_fifo();

Irssi::timeout_add(
  250,
  'poll_commands',
  undef
);

Irssi::timeout_add(
  5000,
  'flush_queue',
  undef
);

Irssi::print("away_bridge loaded");