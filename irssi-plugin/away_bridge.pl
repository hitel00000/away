use strict;
use warnings;

use Irssi;
use IO::Socket::UNIX;
use Time::HiRes qw(time);

our $VERSION = '0.1';

our %IRSSI = (
  authors     => 'Away',
  name        => 'away_bridge',
  description => 'C-001 public/private irssi signal bridge',
  license     => 'MIT',
);

# Prototype path; move to /run later if desired.
my $SOCKET_PATH = "/tmp/away/irc-companion.sock";

#
# -------------------------
# utility
# -------------------------
#

sub event_id {
  return "evt_" . int(time() * 1000) . "_" . int(rand(100000));
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
# transport
# -------------------------
#

sub emit_json {
  my ($line)=@_;

  my $sock = IO::Socket::UNIX->new(
    Type => SOCK_STREAM,
    Peer => $SOCKET_PATH,
  );

  unless ($sock) {
    Irssi::print("away_bridge: relay socket unavailable");
    return;
  }

  print $sock $line . "\n";
  close($sock);
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
# signal handlers
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
# defensive rebind on reload
#

Irssi::signal_remove('message public', 'on_public_message');
Irssi::signal_remove('message private','on_private_message');

Irssi::signal_add('message public', 'on_public_message');
Irssi::signal_add('message private','on_private_message');

Irssi::print("away_bridge loaded");