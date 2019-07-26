v1.4.0
------

Added:

* A calendar that automatically populates with scheduled events! Intended to help users see when events are planned and
  sign up for them if required.
* New Car Details pages! You can now manage your car collection right in Server Manager!
* Car Search! Search for cars by name, tags, BHP, weight, etc. Car search is connected into Quick Races, Custom Races, Championships. Check out the 'Search Help' on any pages with the search bar to find out what kind of things you can search for!
* Skin Upload - you can now upload individual skins on the Car Details page
* Skin Delete - you can now delete individual skins on the Car Details page.
* Improved asset handling - this is a bit behind-the-scenes, but we've made some efforts to make Server Manager's styles (and fonts!) load quicker.
* Car Notes and Download links - you can now add notes and download links to a car.
* Car Tags - you can now add and remove tags to cars. This means if you have a group of cars you use regularly, you can add a tag to them and just search for that tag!
* Improved Content Manager integration! You can now enable a "Content Manager Wrapper" in Server Options, which provides extra information to entrants in the Content Manager server information! If enabled, Content Manager Wrapper shows download links for the cars that you have uploaded (if they have a Download URL set), Championship information positions, and more! As well, the Content Manager Wrapper will make loading server information quicker.
* Added an option to config.yml to use a filesystem session store instead of a cookie session store. This should fix issues that people were having with login not being persisted when running multiple Server Manager instances on the same address but different port. Now, you can specify a different filesystem store for each instance of Server Manager. Check out the config.yml 'http' section for more information on this.
* Added a Content Manager join link to the Live Timings page. This join link can be turned on/off on the server settings page.
* Added a generic welcome message for all drivers on connect, it will also warn the driver if the server is running Sol.
* Server Manager now uses gzip compression where possible. This should improve page load times considerably!

Note, all of the new Car features work best when you have uploaded your cars to Server Manager. If you haven't, the pages will still work, but won't be anywhere near as cool!

Fixes:

* Improved error handling when parsing config.yml, this should give an error with more detail rather than crashing.
* MOTD text will now be automatically wrapped to prevent large horizontal messages on join.
* Fixes a bug where drivers who connect but do not load were left in the Connected Drivers table in Live Timings.
* Live Timings will now reconnect automatically if your connection drops.

v1.3.3
------

**Please back up your data store (as defined in config.yml in 'store' -> 'path') before upgrading to this 
  version!**

Added:

* We have made significant changes to Live Timings in this update, including:
  - A new page layout which should hopefully give more space for the Live Timings table, with the map slightly reduced in size.
  - Live Timings and the Live Map now both use the same source for data, meaning that your browser doesn't need to make as many requests to find out information.
  - Live Timings now use a more standard time format (e.g. 01:23.234 instead of 1m23.234s).
  - Crashes involving Drivers now show the name of the other Driver that was involved in the crash.
  - Track information has been moved into a popover which appears when you click the session title on the Live Timings page.
  - Firefox map resizing bugs are now properly fixed.
  - Various other small bugs are fixed too.
  - A new grid layout for the IFrames on the Live Timings page. On larger screens, you can place two iframes side by side.
  
  This is quite a large change in terms of code. If you find any problems with Live Timings, please let us know and we will sort them out!

* You can now disable DRS Zones for any track in Custom Race / Championship Events. The drs_zones.ini file for the track
  is replaced with a 'no DRS' file, meaning that players can't activate DRS at any point on the circuit. Note: this changes
  actual track files, so if you're using a multi-server setup pointing to the same content folder, this may cause problems
  if you're running races at the same track simultaneously.
* Starting a Quick Race now takes you straight to the Live Timings page.
* Scheduled Championship events now show the start time of individual sessions if applicable.
* You can now explicitly control the Grid/Pit Box position of an entrant in Custom Races and Championships! This is 
  useful if you want to place teammates next to each other in the pits, avoid broken pit boxes or have a custom
  starting grid for a race with no qualifying. It should auto fill sensibly if left alone too!
* Audit logs, Server Manager now locally tracks all actions carried out by registered users. Only admins can access
  the logs, and logging can be disabled in the config.yml. Logs are intended to help server admins track down users
  acting maliciously or just making mistakes, for example deleting a whole Championship an hour before it was 
  meant to start (we're looking at you, Greg).
* Added a link to our new Wiki in the footer! If you want to contribute tips and tricks for Server Manager, the wiki is the place!
  You can access the wiki here: https://github.com/cj123/assetto-server-manager/wiki
* The Server Manager javascript is now minified, which should make the pages load quicker!
* Results tables now use the same time format as Live Timings (e.g. 01:23.234 instead of 1m23.234s).
* You can now split the JSON store into two separate directories: private and shared. This is useful for multiserver setups,
  where multiple instances of Server Manager can share the same database for Championships, Custom Races and AutoFill Entrants.
  Check out the config.yml for more details. Thanks to WraithStar for contributing this!

Fixes:

* Open Championships will no longer empty the team name of a driver that has a team name specified.
* Fixes an issue where tracks with a default layout and an extra layout (e.g. 'wet' tracks) would not be correctly set up
  from Quick Race.
* Users with read access or lower can no longer access replacement championship passwords by exporting the championship.
* Championship overview and add event pages will now warn the user if the selected track has too few pit boxes to accommodate
  the configured number of entrants.
* Changed how process killing is done on Windows, hopefully making stopping plugins more reliable! We've had some mixed results
  on this one, so we'd like to hear if it's working for you now!
* Result tables now hide the Team column if no driver in the results has a team.
* Improved the allowed tyres UI to more clearly show which is enabled and which is disabled.

Removed:

* In an effort to reduce the build size and complexity, we have removed the old Entrant autofill method. This has been
  replaced by the more obvious dropdown in the Entrant box.

v1.3.2
------

**Please note, this release contains breaking changes for run_on_start in config.yml**. If you are using run_on_start,
you need to read the following:

Each run_on_start command is now run from the directory that the binary file is in. 
For example, ./stracker/stracker_linux_x86/stracker --stracker_ini stracker-default.ini now actually performs the following two commands:

1. cd ./stracker/stracker_linux_x86
2. ./stracker --stracker_ini stracker-default.ini

This means that previous configuration entries will need updating! The config.example.yml contains updated examples for how this might work.

Added:

* Plugins are now run from the directory that their executable file is in. Please read the above note for more information.
* Results overviews now show the tyre which was used to complete the driver's fastest lap of that session.
* Added per-Event points breakdowns to Championships!
* Server Logs are now only accessible by users in the "Write" group or above.

Fixes:

* Corrected the sizing of the "Remove IFrame" button on the Live Timings page.
* Corrected the sizing and positioning of the Live Map when the page is resized.
* Added an explanation as to why the UDP ports specified in Server Options do not match the ones in the server_cfg.ini. 
* Fixes a bug where the EntryList was limited to 18 entrants in Custom Races.
* AutoFill entrants are now alphabetically sorted.
* Laps which have Cuts > 0 are now excluded from "Best Lap" in Live Timings
* Fixes misleading times in the Live Timings stored times table by adding leading zeroes to millisecond values under 100ms.

v1.3.1
------

Added:

* Live Map smoothing! Thanks to germanrcuriel on GitHub for this one! It makes a huge difference to the Live Map!
* Removed the gray background from the Live Map, replaced it with a drop-shadow. Thanks again to germanrcuriel for this! 
* Tweaked the layout of the Live Timing description.
* You can now delete AutoFill entrants from the new AutoFill entrants page (available for users with Delete permissions or higher)
* Added Top Speed to Live Timings
* Team Standings are hidden in Championships where no entrant has a team name.
* You can now delete entrants who have registered to a Championship Sign Up Form.

Fixes:

* You can now start Practice Events for Open Championships that do not have any entrants registered to them.
* Championship Sign Up Forms now show how many free slots each car has.
* Championship Sign Up Forms and Open Championships now properly respect the distribution of cars in an EntryList.
  - If a user rejoins an Open Championship in a different car, their original slot in the EntryList is cleared so that
    other Championship Entrants may use it instead. (Previously this slot was removed).
  - Users registering on the Sign Up form can only be put in slots in the EntryList where their requested car matches
    the car defined in the EntryList.
* Fixes a bug where new Entrants could not be added to a Custom Race if all previous Entrants had been deleted from it.
* Fixes a bug where Championship Events with a Second race would sometimes stop running after the first race.
* Fixed an issue where sometimes drivers would not properly disconnect from the Live Map.
* Pickup mode is now enabled for all Championship Practice Events that don't have Booking Sessions.
* The "Locked Entry List" option has a more detailed explanation about how it works.
* Open Championship Events using Booking mode can now be correctly configured. Note that you must create entrant slots in the 
  Championship setup, otherwise the Championship Events will start without any cars available!
* Open Championship Events with Booking mode now have a Booking mode for their practice sessions too.
* The 'Save Race' button at the top of the Custom Race form now saves the race without redirecting you to Live Timings
* Fixes a panic that can occur when using shortened driver names if a driver's name ends in a space.
* Fixes an issue where a driver's initials were not saved correctly when reloading the live map.

v1.3.0
------

Added:

* Added a Championship Sign Up form. When configuring a Championship, you can now enable 'Sign Up Form'. This creates a public 
  registration form where anybody can sign up to be added to a Championship. This has the following options:
  - Require admin approval for applications - Every sign up request must be approved by an administrator before 
    they are added to the EntryList.
  - Ask users for Email - you can request that users enter their email address so you can contact them
  - Ask users for Team - enable a 'Team' field to be filled out by registrants (this is optional though)
  - Let users choose car and skin - On by default, users can pick their car and skin. If turned off, an administrator 
    will have to configure the car and skin in the EntryList once the driver is accepted.
  - Extra Questions - ask registrants to fill out questions that you specify, e.g. Discord username, Twitter handle, 
    number of races completed, etc.
  
  Championship Sign Up requests can be viewed by Write Access users and Approved or Rejected in a new page. 
  This new page also allows Write Access users to email groups of registrants, and download a Comma Separated Values 
  list of the registration requests (to be used in spreadsheets etc.)
  
  We hope that this functionality improves the management of large events for Server Owners! Please get in touch if 
  you run large events and let us know how it goes!
  
* In Server Options you can now configure Server Manager to show the Championship or Custom Race name after the server name
  in the Assetto Corsa server lobby.
* You can now add custom CSS to your server manager install from the Server Options!
* You can now add points penalties to Championship Drivers and Teams.
* Added monitoring and analytics. This will help us keep better track of issues in Server Manager. You can disable this
  in the config.yml.
* Improved 'Reverse Grid' text to explain what happens if the value is set to '1' (a 2nd Race will take place with the grid formed from the 1st Race's Results)
* You can now import Championship results for second races.
* You can now export all of the results for a championship to Simresults together.
* Individual events (custom races and championships) can now override the global server password setting.
* Added the ability to import entire championships.
* Added the ability to use shortened Driver Names in all areas of the Server Manager web UI to protect people's identities online. 
  You can enable this in Server Options.

Fixes:

* Loop Mode is no longer an option in Championship Events
* Imported Championship Events now correctly link to their original results files, meaning that penalties carry across
  to Championships when applied in the results pages.
* Fixes a bug where penalties would not correctly apply to some Championship Events.
* Fixes an issue where Looped Races could hang at the end of a race rather than continuing to the next Looped Race.
* Open Championships will now correctly set up Entrants (and results tables) when importing results files that have new
  Entrants in them.
* Sol session start time should now save properly.
* Locked entry list option should now work.
* Fixes a bug where saving a Championship Event using the top Save button would actually cause the event to be duplicated.
* Reworded the Reverse Grid description

v1.2.2
------

Fixes a bug where new Championship Entrants couldn't be added.

v1.2.1
------

Added:

* Added a MOTD editor
* Added the missing MAX_CONTACTS_PER_KM server_cfg.ini option. We've found this a bit temperamental so use it with caution!
* Ballast and Restrictor are now visible in Results pages
* When adding Entrants to a Custom Race or Championship, the values from the last Entrant in the list are copied to 
  each new Entrant that is added. This should make editing the EntryList a bit easier!
* Championship welcome message now shows a link to the championship overview page (requires server_manager_base_URL 
  option in config.yml)
* Scheduled events now show their times in your local timezone.
* You can now subscribe to iCal feeds for scheduled races at a server level or per Championship.

Fixes:

* Limited the Live Map refresh interval to a minimum of 200ms. It is now 500ms by default in config.yml.
* The Manage Event button in a Championship is now visible for completed events for admin users only. This
  should allow you to import results files if Championships fail to complete successfully.
* Starting a Custom Race now takes you to the Live Timings page.
* Servers with really long names now have the name truncated in the navigation. You can hover over the name to see the full text.
* Fixed an issue where lots of UDP errors would appear in the log.
* Championship Name is now a required field
* Removed a non-critical error message from the logs
* Fixed live map extra data toggle.
* Detected improper disconnects (game crashes, alt+f4 etc.) in live timing.
* Fixes an issue where configured car skins would be lost in Open Championships.

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
* Adds skin, ballast and restrictor customisation for individual Championship Events. You can overwrite these options
  for all Championship Events in the Edit Championship Page.
* Added configurable IFrames to the live timings page. Users with write access can modify and add IFrames to the
  page (they will persist for all users between events). Intended for use with event live streams or track info etc.
* Added extra track info to live timings page.
* Added an extra info pane to drivers on the live map that displays their current speed, gear and rpm. This can be
  toggled on/off by clicking their name in the live timings table.
* Changed the layout of the live timings page to better accommodate the new features.
* Added "Import Championship Event" functionality, which lets you import non-championship results files into a
  championship. To use this, create a championship event for the track and layout you wish to import results to. Then,
  click on "Manage Event" on the Championship page and select the session results files to import from.
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
* Vastly improved Championship points scoring. Points scoring now adheres to the following rules:
  - If a driver changes car but NOT team or class, both team and driver points persist.
  - If a driver changes team, but NOT class, drivers points persist, team points stay at old team and new points 
    earned go to new team. You can override this by turning on the "Transfer Points from previous team?" switch when you
    change a driver's team name.
  - If a driver changes class, an entirely new entry is made but the old one is not deleted - new points earned go to the 
    new team and new driver entry.
  
  A byproduct of this is that once points have been added to a Championship Class, they cannot be removed. That is, if you
  have 6 drivers in a Championship Class and you remove 2, there will still be 6 points inputs in the Class. This is so
  that previous Championship Events have the correct number of points for their calculations.
* Added logging to server-manager.log - this should make debugging issues easier.
* Moved "Result Screen Time" option to Custom Race / Championship Event configuration instead of Server Options
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
* Fixes an issue where Second Race Points Multiplier would default to 0 if not specified, rather than using 1.
* We now exclude disqualified drivers from points in the race they were disqualified from.
* Championship Events now show the cars that entered the race or are due to enter the race in their header, rather
  than just showing the cars entered into the Championship.
* Added logging to server-manager.log - this should make debugging issues easier.
* Improved reliability of live timing table.
* Event scheduling now uses your local timezone from your browser.
* Fixes incorrectly decoded utf32 strings coming through UDP data.
* Booking Mode now works correctly. Closed Championships have booking mode disabled for all events.

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

