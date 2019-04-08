v1.2.0
------

Note: This update changes how the accounts work, you will need to re-add all of your existing accounts in the server
control panel! To do this, you will need the new default admin details:

  * username: admin
  * password: servermanager

We also recommend backing up your data store (as defined in config.yml in 'store' -> 'path') before upgrading to this 
version!

Now, on to the changes!

Added:

* Account Management via the web interface! No more fiddling with configuration files! You will need to re-add your accounts
  in the web UI.
* Adds Fixed Setups to Custom Races and Championships. Fixed setups can be uploaded on the Cars page. You can fix
  a setup for a whole championship or individually for specific events in the Championship.
* Adds skin, ballast and restrictor customisation for individual Championship Events.
* Added configurable IFrames to the live timings page. Users with write access can modify and add IFrames to the
  page (they will persist for all users between events). Intended for use with event live streams or track info etc.
* Added extra track info to live timings page.
* Added an extra info pane to drivers on the live map that displays their current speed, gear and rpm. This can be
  toggled on/off by clicking their name in the live timings table.
* Changed the layout of the live timings page to better accommodate the new features.
* Added "Import Championship Event" functionality, which lets you import non-championship results files into a
  championship. To use this, create a championship event for the track and layout you wish to import results to. Then,
  click on "Manage Event" on the Championship page and select the session results files to import from.
* Added important links to championship create/edit. You can now add important links to track/car downloads etc.
* Added car images to Championship pages.
* Added car info to live timing table
* Added an option to only upload official ks content from a folder
* Added option to upload multiple content folders by dragging them into the drag and drop upload boxes.
* Added a more informative message for users who experience issues launching Server Manager. We're trying our best
  to make Server Manager really good, and we're a little disheartened by negative reviews caused by people not managing
  to follow our setup instructions, so hopefully this will help with debugging!
* Added a dropdown to the Entrant box which makes auto fill much more obvious and less likely to be interfered with
  by browsers.
* Added a "Delete" group - they are the only group (other than admin) allowed to delete content, championships, races, etc.
* You can now change which assetto server executable file is run by Server Manager. By default, acServer(.exe) is used.
  See config.yml "executable_path" for more information. This means tools such as ac-server-wrapper should now be
  compatible with Server Manager! You can even write your own wrapper scripts around acServer if you'd like.
* Added buttons to change the Championship event order and show/hide completed/not completed events.
* Looped practice events will now persist drivers in the live timings table across each event.
* Added a text input (with support for images, embedded video etc.) to Championship pages. Intended for adding information
  about the championship, rules, links to content used etc.
* Added missing "MAX_CONTACTS_PER_KM" to server configuration options.
* Added disconnected table to live timing page, shows best times of disconnected drivers from the current session.
* Added blacklist.txt editor.

Fixes:

* Fixes an issue preventing the upload of older cars which contain a data folder rather than a data.acd file.
* Removed unnecessary duplication of entrants on Championship pages.
* Fixes an issue with illegal Byte Order Marks preventing some track info files from being read.
* Fixes an issue where some Live Map cars would not properly clear on server restart.
* Fixes an issue where new entrants in a Championship were not persisted for autofill.
* Fixes an issue that made it impossible to start quick/custom races on mobile safari.
* Fixes an issue where Championship Events were not correctly finished/tracked.
* Added logging to server-manager.log - this should make debugging issues easier.
* Improved reliability of live timing table.

v1.1.3
------

* Fixes an issue with Championship Practice Events not working after updating the cars in the Championship entry list.

v1.1.2
------

* Adds support for sTracker. Read the new config.yml file for more information on setting up sTracker.
* Adds support for running processes alongside the Assetto Corsa server. Each process is run when the server
  is started, and killed when the server is stopped. See config.yml for more information.
* Improves UDP forwarding to only forward as many bytes as were received.
* Log outputs are now limited to a size of 1MB. When the log output reaches 1MB, it is trimmed to keep the most recent
  messages.
* Championships are now split into active and completed championships. They are ordered by the time they were last
  updated.
* Fixes a bug where tyres configured in a championship event would not carry across to the next championship event or
  load into the edit championship event page.
* Fixed scheduled events for time zones outside of UTC.
* Improved some page layouts on mobile devices

v1.1.1
------

* Fixed a bug that caused some scheduled races to not start correctly.

v1.1.0
------

We recommend re-uploading all of your tracks after doing this update! Some new features will only work with
new track assets!

Please also consult the config.yml in the zip file, there is a new section: "live_map" that you must add to your
config.yml to get live map functionality!

* Added a Live Map. You'll need to re-upload your tracks to see the live map, since it requires new
  track assets.
* Added support for 'Reverse Grid Positions' races within Championship events. If a second race occurs, the championship
  page will show results for that too. It will correctly add points to the entrants and optionally can apply a
  multiplier to all second races to scale their points. This multiplier can be a decimal, and can even be negative!
* Added the ability to schedule championship events and custom races.
* Added button on results page to open the results on the SimResults website.
* When creating a race the number of available pit boxes for a track/layout is now displayed, max clients is limited to
  this number (requires manual upload of track - including default content).
* Championship events now welcome each player with a message describing their current position in the championship
  and who their nearest rivals are.
* Improve handling of tracks which have a default layout (i.e. data folder in the base of the track directory) AND extra
  layouts. This fix adds compatibility for mods such as the Assetto Corsa Wet Mod.
* Added support for plugins such as KissMyRank. Follow the KissMyRank setup, but instead of editing
  server_cfg.ini, edit the Options in Server Manager (it overwrites server_cfg.ini!)
* Overhauled UDP proxying to work with sending messages as well as existing support for receiving.
  (This is what makes KissMyRank etc work!)

v1.0.2
------

* Increase number of results per result listing page to 20.
* Add a 404 error for results pages that don't exist
* Results listing page now shows 10 pages in the pagination bar with options to skip to the front and end,
  and forwards/backwards by 10 pages
* Fixed an issue with named Custom Race entrants losing their car/skin on race start
* Collision speeds on Live Timings page are now rounded to 2 decimal places

v1.0.1
------

* Fixed an issue with populating default points when creating championship classes. Points for places beyond the F1
  defaults now show '0' when created, rather than '25' which was incorrectly shown before.
* Average Lap Time on Results pages is now calculated lap times that meet the following conditions:
    "if lap doesnt cut and if lap is < 107% of average for that driver so far and if lap isn't lap 1"
* Fixed an issue with Quick Race Time/Laps selector not defaulting to the correct value.

v1.0.0
------

Initial Release!

