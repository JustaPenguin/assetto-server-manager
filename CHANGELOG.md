v1.7.3
------

Added:

* ACSR Ratings are now shown in ACSR enabled Championships, in Driver Standings, the Entrants tables and the Sign Up form responses (including the CSV export).
* You can now download Server Logs from Server Manager (thanks @mazzn!)
* You can now run Open Championships with ACSR integration!
* Server Manager now logs the output of the acServer executable to the logs folder inside your Server Install Path. You can configure this in Server Options -> Miscellaneous
* In a Multi-Server environment, you can now set different account groups per server, for example you could set an Account to have "Write" access on one server and "Delete" access on another.
* Added a new permission - "No Access" - which blocks a user from doing anything on a server.
* Open weather API Lua plugin can now automatically find the location of the track (so long as the track json file contains this information) and set the weather accordingly! Thanks to @mike855 for this contribution!

Fixes:

* On Windows, batch files are now stopped correctly by Server Manager.
* Fixes an issue where Championship Class IDs could overlap causing Multi-class championships to incorrectly report standings.
* Fixes an issue where Championship and Race Weekend exports on Firefox would not export with their full name.
* Fixes an issue where Championship/Race Weekend sessions could not be cancelled if the server is not running.
* Fixes an issue where Race Weekend Race sessions would have their wait time forced to 120 seconds.
* Fixes an issue where the progress of a Championship with Race Weekends would be incorrectly reported on the Championship List page.
* Fixes an issue where Championship attendance would not work correctly for event with Race Weekends.
* Fixes an issue where watching content folder for changes could cause a crash on Windows.
* The Championship Entrant table is now sorted alphabetically.
* Championship Event Pitbox overrides are now applied correctly on Championships with Sign Up forms enabled.
* Fixes an issue where Sol dates could be set to dates before 1st January 1970, leading to a Shaders Patch crash on game launch. Dates before 01/01/1970 are now set to 01/01/1970.

Removed:

* Removed max limit of Damage Multiplier (was 100%). Happy crashing!

v1.7.2
------

Added:

* KissMyRank integration! We've even made it so you can use both sTracker and KissMyRank! Go to the new "KissMyRank" page to find out how to set up KissMyRank. We're marking this as "beta" currently. If you find any issues - please report them to us!
* In a Multiserver setup Auto Loop events are now per-server! You can loop different events on different servers, only have one looping server etc.
* You can now duplicate any Championship Event (including Race Weekends!)
* The Championship Sign Up form now allows multiple Steam GUIDs separated by a semi-colon (for driver swaps).
* You can now set a timer to forcibly stop a Custom Race after a certain time period. This is intended to allow servers to rotate through looped events every x minutes regardless of players being active on the server. The timer can be set to not forcibly stop the server if drivers are online.
* Server Manager now automatically sets up the sTracker config path and server folder path.
* You can now manually specify the IP address used by the Content Manager Wrapper. This fixes issues with IPv6 incompatibilities (thanks @mazzn!)
* Search Index Improvements - the folder name of the cars is now included in the search index. This should help yield more results for searches. Note that you will have to rebuild your search index (at the bottom of the Server Options page) for this to take effect.

Fixes:

* Fixes an issue where car setups with invalid ini keys would not upload properly.
* Fixes an issue where AutoFill Entrants would not be saved when editing a Custom Race
* Championship Race Weekends now display all sessions in the correct order
* Fixes issues where stopping the acServer process could cause Server Manager to lock up or crash

v1.7.1
------

Fixes:

* Fixes an issue where ACSR enabled Championships could cause Championship Event points to be incorrect. If your Championship points are incorrect, re-import the results files in the Championship Event ("Manage Event" -> "Import Results")
* Fixes an issue where Server Manager sometimes would not detect and handle acServer shutdown correctly.

v1.7.0
------

### ACSR

This release adds support for Assetto Corsa Skill Ratings (ACSR). ACSR is a new service for organising and taking part in Championships through Assetto Corsa Server Manager. 

When you set up a Championship in Server Manager, you will be given the option to "Export to Assetto Corsa Skill Ratings". This publishes your Championship to the ACSR Championships List page, so that drivers can find your Championship and sign up to it!

* You can view more about ACSR here: https://acsr.assettocorsaservers.com
* For help configuring ACSR, check out our wiki: https://github.com/JustaPenguin/assetto-server-manager/wiki/ACSR-Configuration

ACSR requires Server Manager Premium. You can purchase Server Manager Premium by following the instructions in the free version of Server Manager.

### Server Manager

Added:

* Admins can now export full Championship information, including Sign Up Form responses.
* Exported Championships and Race Weekends now export as JSON files to download, rather than showing JSON in the browser.
* Improved Discord message formatting (thanks @cheesegrits!)
* It is now possible to specify a Race Weekend's initial Entry List as the parent for a session. You can also now use a grid offset when filtering entrants from this entrylist.

Fixes:

* Improved the logic of "fastest lap across multiple results files" sorting in Race Weekends whilst using fallback results sorting (premium)
* Fixes an issue where the server name could not be clicked to go to the homepage on mobile devices
* Fixes an issue where Championship Driver and Team penalties would be lost when editing a Championship.
* Drastically improved the speed of STracker pages through some reworking of how STracker is proxied.
* Fixed an issue where some STracker links would not work correctly.
* Fixes an issue where car skins with special characters in their name would not display in the race setup page and Championship Sign Up form.
* Fixes issues with download links in Discord messages (thanks @cheesegrits!)
* On initial server setup, the admin account is only created if one doesn't exist.
* The JSON Store now correctly sets the updated time of Custom Races, Championships and Race Weekends.

---

v1.6.1
------

**Please note, this release deprecates use of "run_on_start" in config.yml. Please read the new config.yml "plugins" section if you were using run_on_start!**

Added:

* Added send chat to driver option to the admin panel on the Live Timings page.
* The UDP message receiver now detects if it has fallen behind while handling messages. If it has, it reduces the refresh rate that the Assetto Corsa Server sends messages at, so it can catch up. If you see a lot of "can't keep up!" messages in the Server Logs, you probably need to increase your 'refresh_interval_ms' in the config.yml.
* Added configurable Open Graph images to both the manager as a whole and championships, with these you can link to images that will be shown whenever you share a link of the manager/championship pages on social media (premium).
* Optimised the handling of UDP messages to improve performance.
* When using "Any Available Car", one of each car is added to the EntryList (so long as there are enough entrants!) and then after that Entrant cars are randomised.
* Added a new 'plugins' section to config.yml. Please use this instead of 'run_on_start'. This has been added to fix issues with spaces in plugin path names.

Fixes:

* Fixed an issue with filtering from results files in Race Weekends when the Race Weekend is within a Championship.
* We've changed the method that we use to check if scheduled events need running. Hopefully this should make scheduled events more reliable!
* Fixes a memory leak when proxying STracker pages

---

v1.6.0
------

Added:

In this update we've added the following premium features:

* Multiserver support! This is a much-requested feature, and we've finally found a way to deliver it! Premium builds of Server Manager now come with an extra tool: the multiserver manager.

  - The multiserver tool handles installation and setup of however many servers you want!
  - It works on Windows and Linux
  - Custom Races, Championships, Race Weekends and all your uploaded Content are shared between each server!
  - Accounts and login are shared between each server!
  - You can switch servers using the new Server Switcher in the top right hand corner of your Server Manager page.
  - The Server Switcher also shows you what kind of events are running on each server, and how many people are connected!
  - The multiserver tool handles updating server-manager for you!
  
  When you buy Server Manager Premium, the multiserver tool will be included alongside server manager. Please check out the README for more details and a setup guide!

* Lua Plugin hooks! You can now run custom scripts with a bunch of hooks for Lua (event start, results load and more!), have a look in server-manager/plugins for some examples. (there's a nice readme guide in there too!). If you want to enable one of the three Lua examples before just uncomment them in the lua files!
* Weather API with Lua plugins! As a nice example of what you can achieve with Lua plugins we used one to implement a weather API!
* Auto ballast based on championship position with Lua plugins! Another Lua example, this one applies a ballast to each driver in a championship when an event is started based on their championship position.
* Auto collision disqualifier! This is our last Lua example, it disqualifies drivers if they exceed a certain number of collisions or have a collision over a certain speed.
* Added an option in Race Weekends to filter from the best lap or number of laps completed across multiple results files.

If you don't have premium yet, you can get it by following the instructions on Server Manager!

As well as our premium features, we have the following additions...

* Results pages now show your 'Potential' lap time - the sum of your best sectors.
* Added an option to the config.yml to watch the content folder for changes and keep the car search index updated.
* Added an option to the config.yml to prevent server manager from opening a browser on Windows on start.
* Accounts are now part of the shared JSON store. This means if you were running split JSON stores using the 'shared_data_path' config.yml variable, you will need to re-set up server manager accounts (i.e. by copying them into the shared JSON store)
* You can now import existing Custom Races and Race Weekends to Championships.
* You can now upload results files through the Server Manager UI on the results page.

Fixes:

* Fixed an issue with scheduled Championship events that could clear all of the other scheduled events on the Championship when they started.
* Setting the ballast on an entrant to a larger value than the Max Ballast option no longer stops the server from starting.
* Results pages now correctly display statistics per car - so if you're switching cars in a session you can see accurate reports for that car.
* Penalties are now applied per driver and car, rather than just per driver.
* Fixes an issue where events scheduled in a multi-server scenario would start on the wrong server.
* STracker proxy plugin local/plugin ports should now be properly set by the Server Manager UI.
* Tyres with spaces in their short_names cause the server to fail to connect to lobby, stopped those tyres from being uploaded to the server.
* Prevent users from being able to set up a race that will cause the server to crash by setting pickup mode off, locked entry list on and reverse grid race to anything other than 0.
* Fixes an issue where a missing ui_track.json file would cause a track page to load incorrectly.
* Fixed an issue where Championships sometimes could not be edited.
* Fixed date/time formatting for Championship event session start times.

---

v1.5.3
------

Added:

* Live Timings "Stored Times" will now persist between reboots of Server Manager, if the next event started is the same as the last event running before the Server Manager reboot.
* You can now configure Server Manager to restart any Event that was running before Server Manager was stopped. Check out "Restart Event On Server Manager Launch" in Server Options.
* Added an option to prevent web crawlers from accessing any pages on the manager using a robots.txt.
* Added information about whether a car/track is default/mod/DLC to list and detail pages.
* Championship and Race Weekend Looping Practice Sessions are now labelled in the navigation bar, e.g. "Championship Practice in Progress"
* Added penalty options to Qualifying and Practice sessions, penalties can be applied independently in each session. In Race Weekends these penalties will affect the entry list of the next session.
* Added a button to blacklist a driver directly from results pages, the button is inside the penalties popup and can only be accessed by admins.
* Added content download links to discord messages (thanks @cheesegrits!)
* Enabled Content Manager Wrapper "Install Missing Content" button, just add links to tracks/cars on their detail pages and the button will work for events using that content!
* You can now filter stock, DLC and mod cars in or out of the car search. Check out the car search help for more details! Please note that you will need to rebuild your search index for this to work. Go to Server Options, scroll down to "Maintenance" and click "Rebuild Search Index"!
* Server Manager will now set up some example Championships and Custom Races if you have not yet created any
* You can now sort the EntryList for a Championship Race Weekend Session by the number of Championship points a driver has. This could be useful for running reverse grid qualifying races!
* Added a health-check endpoint. Hopefully this will help us with debugging issues!

Fixes:

* Fixes track/car display in dark mode
* Fixes track details page names to be a bit nicer
* Added Black Cat County to the list of default content
* Fixes an issue where Server Manager would not start when a recurring race with an end date had a scheduled recurrence while Server Manager was offline.
* Custom Races are now unscheduled when they are deleted.
* Stopped users from being able to delete their own accounts.
* Fixes an issue where drivers who switched teams mid-Championship had an incorrect number of races shown for their teams.
* Championship event inheritance now correctly uses the previous event setup, not the first event setup
* Fixes an issue where tyres did not show correctly in Session configuration for Championship Race Weekend events.
* Fixes an issue where placeholder entrants were incorrectly added to the entrylist of a Race Weekend practice session.

---

v1.5.2
------

Added:

* Added information about whether a car/track is part of a DLC or a Mod when creating an event.
* Discord Enhancements (Thanks @cheesegrits!):
  - Splits the '!schedule' command into '!sessions' (full "wall of text" individual session calendar, restricted to one week ahead) and '!schedule' (abbreviated, one per race calendar). This still needs work, as can easily exceed Discord's max msg length.

  - Added role mentioning. If the optional DiscordRoleID is set, that role will be mentioned in all Discord notifications (meaning anyone with that role will get pinged). Additionally, if the optional 'DiscordRoleCommand' is also set, we will attempt to add/remove that role for users, when they issue the "!whatever" command verb - this requires that the bot has "Manage Roles" permission.

  - Changed NotificationReminderTimer to NotificationReminderTimers (plural), to support comma separated multiple timers (like "90,15" for two reminders at 90 and 15 minutes).

  - Added option to disable notifications when a race is scheduled.

  - Added notification for scheduled races being cancelled.

  - Improved formatting of Discord messages, everything is now an embed (except the role mention, which has to be part of the basic message).

Fixes:

* Fixes track pages for users running Server Manager on Windows
* Fixes an issue where Championships with 'Any Car Model' specified would fail to find a class for a car.
* Fixes an issue where cars with no skins might prevent a race from starting.
* Fixes an issue where Scheduled Championship Race Weekend sessions caused the calendar to error on load.
* Fixes the Race Weekend "Start after previous session" checkbox not displaying correctly.
* Fixes an issue where all drivers were incorrectly disconnected from the Live Timings page when an event with multiple sessions looped

---

v1.5.1
------

Added:

* Added a "Start on Tyre from Fastest Lap" option to Race Weekend Filtering. You can use this to force an entrant for a session to start on the tyre they used to complete their fastest lap in the previous session. This might be useful when simulating F1-style qualifications.
* Added a "Start after Parent Session has completed" option to the Race Weekend Schedule popup
* Added configurable negative points modifiers for championships for crashes and cuts.
* Added an Entrant Attendance field to Championship table to help admins keep track of who is showing up for races.
* Enabled recurring events within championships. A recurring event inside a championship will create a copy of itself in the championship list, scheduled for the next time as defined by the recurrence rule.
* Added "Register with Steam" to Championships with a Sign Up Form.
* Added "Sign in with Steam" to the Update Account Details page.
* Added collision maps to result pages so you look back on your numerous incidents with clarity. On the event tab you can control which collisions are shown on the map.
* Added track info pages that work much the same as car info pages.
* STracker integration! This is still somewhat experimental, please report any bugs you find with this! Check out the STracker Options page to get started.
* Added an "Any Available Car" option to Entrylists. In Sign Up Championships, this allows you to select a range of cars and let registrants choose which car they want. If not filled, these car slots become random car assignments in Custom Races.

Fixes:

* Fixes an issue where you could not create a Championship with a single entrant.
* Skins with a # in their name no longer break the car details page.
* Fixes an issue where the data file for some newer Assetto Corsa cars could not be read.
* You can now assign negative points penalties in Championships (so you can add points to people in the Standings if you want!)
* Fixed a couple of issues with plugins running with the Assetto Process.
* Championship entrant/standings tables can now overflow and scroll if they get really long.
* Improved fastest lap sorting in Championships.

---

v1.5.0
------

Added:

* Race Weekends (premium feature) - A Race Weekend is a group of sequential sessions that can be run at any time. For example, you could set up a Qualifying session to run on a Saturday, then the Race to follow it on a Sunday. Server Manager handles the starting grid for you, and lets you organise Entrants into splits based on their results and other factors!
  
  - You can create individual Race Weekends or create Race Weekends as part of a Championship
  - Race Weekends need a fully configured Entry List to work properly. If you're using a Championship Race Weekend, the Championship Entry List is used.
  - You can add as many sessions to a Race Weekend as you like! You could run 4 Practice sessions, followed by 3 Races, and then a Qualifying, then a Practice, another Race, etc! You have full control!
  - You can start individual Race Weekend sessions at any time. So you can run one session one day, then another one three weeks ahead if you like. We think this will be useful for things such as Endurance events, where you maybe want your drivers to qualify on a different day so they don't tire themselves out before doing a 2 hour race.
  - By default, the results of a session will form the Grid positions for the next session.
  - You can sort the results of a session by a number of different factors (fastest lap, total race time, collisions, cuts, safety, random and alphabetical)
  - All session results can be reversed in the same way you can configure a Reverse Grid Race normally.
  - To manage the flow between sessions, click on the arrow between two sessions.
  - A session grid can be formed from the results of multiple parent sessions. This means you can split and merge the Race Weekend as much as you like. For example, you could set up an F1-style qualifying using Race Weekends! (Check out the example Race Weekend).
  - Race Weekend sessions are shown in a flow-chart, so you can see the connections between the sessions.
  - In Multiclass Championship Race Weekends, the sorting of the Entry Lists is per class. The classes are then ordered by the fastest lap of each class (so LMP1 cars will be ahead of GTE cars on the grid, for example)
  - Each Championship Race Weekend session allows you to set points for the individual session. Championship points are calculated using these points values.
  - You can schedule individual Race Weekend Sessions too!

* Discord integration! Thanks to @cheesegrits for this! Check out the Server Options page for more information.
* Dark Theme! You can now set Server Manager to use a Dark Theme in the Server Options. You can also edit this for your account in the "Update Details" page.
* A re-ordered homepage with the tiles sorted into categories.
* Server Name Templates - you can now specify (in Server Options) how Server Manager displays your server and event name.
* We've tidied up the Server Options page a bit. It was getting quite large! The new headings should make things a bit more readable.
* If 'Show Race Name In Server Lobby' is enabled, Quick Races and Custom Races with no specified name now show the track name in the Server Lobby.
* A global option to switch speed units from Km/h to MPH for people who want to use the correct measurement system.
* "Force Virtual Mirror" now defaults to on in all race setup forms.
* Admin Control Panel on the Live Timing page. Allows admins to send server wide messages, advance to the next/restart session, kick users and other admin commands!
* You can now re-order Championship Events! Drag them around on the page using the top bar of the Championship Event.

Fixes:

* Championship Sign Up Forms are only shown if the Championship has free slots left and the Championship is not fully complete.
* Championships now always show the 'Entrants' tab, so you can check to see if you're in the list!
* Improved cache validation so that user-uploaded files can change without needing to empty the browser cache.

---

v1.4.2
------

Added:

* Added configurable server join and content manager messages. Check out the "Messages" page for more details.
* Championship Events now show the best qualifying and fastest lap in each class.
* You can now rename drivers on results pages.
* Drivers can now add their Driver Name, GUID and Team to their account. This will highlight their results on all result pages, add them to the autofill entrant list and automatically input their information if they sign up for an event with a sign up form
* Added a "Persist Open Championship Entrants" option. When turned off, this prevents the Championship Entry List from filling up, so you can run multiple Championship events in a quick-fire fashion without needing to edit the Entry List between each one. (Championship Points will continue to work correctly).
* Championship Sign Up Forms now replace your existing sign up with a new one if you sign up with the same GUID and email, so you can request to change cars.

Fixes:

* Results in the results listings now only show entrants that took part in a session, rather than all registered entrants for that session.
* Added a popup to alert admins when there are pending Registration Requests for a Championship.
* Added a "Back to Championship" button on the Championship Registration Request management page.
* The Championship EntryList displayed before events start now shows the car skins.
* Championship Entrants are now shown in the order of their pitboxes in the Championship Entrant overview table (shown only when championship progress is 0%).
* Fixed an issue where registered users did not have their skin set up on individual Championship Events.
* Fixes an issue where search indexing could fail due to a car with malformed details JSON. Cars which have malformed details JSON (that we can't clean up) will now load without any car details, allowing search indexing to continue.
* Fixes an issue that caused incorrect ordering of Multiclass Open Championship results. If your Multiclass Open Championships have incorrect classes, you can re-import the Championship event results to fix the issue. 
* Fixes an issue where fastest laps were only awarded to the fastest class in a Multiclass Championship.
* Fixes an issue where Sol session start times would change to an incorrect time when editing an event.
* Locked Entry List and Pickup Mode options are now available to be changed in non-Championship events. Do with them what you will.
* Fixes an issue where Championship Sign Up forms couldn't assign a car when users were not allowed car choice.

---

v1.4.1
------

Fixes:

* Added the "Sleep Time" option to the Server Options page. Added a migration to set the Sleep Time option to 1 for all users.
  This fixes an issue where it was not correctly set to 1 (a value that kunos explicitly recommends), which on smaller servers could cause 100% CPU usage.

---

v1.4.0
------

Added:

* A calendar that automatically populates with scheduled events! Intended to help users see when events are planned and
  sign up for them if required.
* New Car Details pages! You can now manage your car collection right in Server Manager!
* Car Search! Search for cars by name, tags, BHP, weight, etc. Car search is connected into Quick Races, Custom Races, Championships. Check out the 'Search Help' on any pages with the search bar to find out what kind of things you can search for!
* Scheduled Race recurrence! You can now set scheduled races to recur at regular intervals.
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
* Added server "Performance Mode" option to config.yml. If this mode is enabled server manager will disable live timings completely, reducing cpu utilisation. This setting may be used in the future to disable further advanced options in order to improve performance.
* You'll now see this Changelog in Server Manager once per account every time you upgrade. You can also view the Changelog in Server Manager itself at any time using the link in the footer!

Note, all of the new Car features work best when you have uploaded your cars to Server Manager. If you haven't, the pages will still work, but won't be anywhere near as cool!

Fixes:

* Improved error handling when parsing config.yml, this should give an error with more detail rather than crashing.
* MOTD text will now be automatically wrapped to prevent large horizontal messages on join.
* Fixes a bug where drivers who connect but do not load were left in the Connected Drivers table in Live Timings.
* Live Timings will now reconnect automatically if your connection drops.
* Only upload official ks content is now working again!
* Fixes an issue where Open Championship EntryLists would not be correctly preserved when assigning car slots to pre-existing Entrants. 
* Added a server wide fallback sorting option for events where AC has outputted an incorrect driver order in the result json file. Only enable this if you have sorting issues. If you have championship events where the sorting is incorrect you will need to re-import the results files using Manage Event, Import Results.
* Fixes an issue where the sessions "Time" / "Laps" selector did not show an input field when loading a previously saved race setup.
* Some errors which were being seen often are now 'warnings' not errors, so you won't see them as often.
* Reworked the Live Timings table to perform better and prevent scrolling issues.
* Removed the strict frontend sorting of pit IDs when creating an event. Now you can put cars wherever you like, but they will then be automatically sorted based on weighting. E.g. 0-3-5-5-6 becomes 0-1-2-3-4. Please try to avoid multiple entrants with the same pit ID, as their pitbox will essentially become random.
* Entrants in the autofill list should no longer duplicate when using the json store, although you will need to manually remove any existing duplicates.

---

v1.3.4
------

Fixes:

* Fixed an entry list issue that made some cars impossible to connect to if the pit ID selection had a gap in it. Any gaps in the pit IDs will now be closed automatically (e.g. 1-2-4 becomes 0-1-2). If you want gaps in your entry list please add dummy cars to it.

---

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
  You can access the wiki here: https://github.com/JustaPenguin/assetto-server-manager/wiki
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

---

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

---

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

---

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

---

v1.2.2
------

Fixes a bug where new Championship Entrants couldn't be added.

---

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

---

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
  
  - A byproduct of this is that once points have been added to a Championship Class, they cannot be removed. That is, if you
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

---

v1.1.3
------

* Fixes an issue with Championship Practice Events not working after updating the cars in the Championship entry list.

---

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

---

v1.1.1
------

* Fixed a bug that caused some scheduled races to not start correctly.

---

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

---

v1.0.2
------

* Increase number of results per result listing page to 20.
* Add a 404 error for results pages that don't exist
* Results listing page now shows 10 pages in the pagination bar with options to skip to the front and end,
  and forwards/backwards by 10 pages
* Fixed an issue with named Custom Race entrants losing their car/skin on race start
* Collision speeds on Live Timings page are now rounded to 2 decimal places

---

v1.0.1
------

* Fixed an issue with populating default points when creating championship classes. Points for places beyond the F1
  defaults now show '0' when created, rather than '25' which was incorrectly shown before.
* Average Lap Time on Results pages is now calculated lap times that meet the following conditions:
    "if lap doesnt cut and if lap is < 107% of average for that driver so far and if lap isn't lap 1"
* Fixed an issue with Quick Race Time/Laps selector not defaulting to the correct value.

---

v1.0.0
------

Initial Release!

