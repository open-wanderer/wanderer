// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Hungarian (`hu`).
class AppLocalizationsHu extends AppLocalizations {
  AppLocalizationsHu([String locale = 'hu']) : super(locale);

  @override
  String get biking => 'Biking';

  @override
  String get canoeing => 'Canoeing';

  @override
  String get walking => 'Walking';

  @override
  String get about => 'A programról';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Ön most a profilját készül törölni. Minden nyomvonala törlődik. Szeretné folytatni?';

  @override
  String get account_privacy => 'Account privacy';

  @override
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Activities',
      one: 'Activity',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => 'Add Bio';

  @override
  String get add_entry => 'Bejegyzés hozzáadása';

  @override
  String get add_to_list => 'Hozzáadás a listához';

  @override
  String get add_waypoint => 'Útvonalpont hozzáadása';

  @override
  String get added_trail_to => 'Hozzáadott nyomvonal a';

  @override
  String get added_trails_to => 'Hozzáadott nyomvonalak a';

  @override
  String get after => 'After';

  @override
  String get all => 'All';

  @override
  String get all_activities => 'All activities';

  @override
  String get allow_auto_geolocate =>
      'Begin drawing a new trail from your current location';

  @override
  String get alphabetical => 'Betűrendben';

  @override
  String get already_account => 'Már rendelkezik fiókkal?';

  @override
  String get altitude => 'Magasság';

  @override
  String get amenity => 'Amenity';

  @override
  String get api_documentation => 'API Dokumentáció';

  @override
  String get api_tokens => 'API Tokens';

  @override
  String get api_tokens_hint =>
      'API Tokens can be used to grant 3rd party applications access to your wanderer account.';

  @override
  String get apply_user_settings => 'Apply user settings';

  @override
  String get attraction => 'Attraction';

  @override
  String get author => 'Author';

  @override
  String get avatar => 'Avatar';

  @override
  String get average_speed => 'Avg. Speed';

  @override
  String get avoid_bad_surfaces => 'Avoid Bad Surfaces';

  @override
  String get back => 'Back';

  @override
  String get back_to_login => 'Back to Login';

  @override
  String get bakery => 'Bakery';

  @override
  String get barrier => 'Barrier';

  @override
  String get basic_info => 'Alap információk';

  @override
  String get basque => 'Basque';

  @override
  String get before => 'Before';

  @override
  String get behavior => 'Behavior';

  @override
  String get bicycle_parking => 'Bicycle Parking';

  @override
  String get bicycle_rental => 'Bicycle Rental';

  @override
  String get bicycle_shop => 'Bicycle Shop';

  @override
  String get bike_type => 'Bike Type';

  @override
  String get bus_stop => 'Bus stop';

  @override
  String get by => 'by';

  @override
  String get campsite => 'Campsite';

  @override
  String get can => 'can';

  @override
  String get cancel => 'Mégsem';

  @override
  String get car => 'Car';

  @override
  String get car_motorcycle => 'Car/Motorcycle';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Kártyák',
      one: 'Kártya',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Kategóriák';

  @override
  String get category => 'Kategória';

  @override
  String get change => 'Változás';

  @override
  String get change_email => 'Change email';

  @override
  String get change_password => 'Change password';

  @override
  String get changelog => 'Változás napló';

  @override
  String get chinese => 'Kínai (egyszerűsítve)';

  @override
  String get clear_all => 'Clear all';

  @override
  String get climbing => 'Climbing';

  @override
  String get close => 'Close';

  @override
  String get collapse_trail_list => 'Collapse trail list';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Megjegyzés',
      one: 'Megjegyzés',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Teljesítve';

  @override
  String get completed_a_trail => 'completed a trail';

  @override
  String get completed_tours => 'Completed tours';

  @override
  String get completion_status => 'Befejezés állapota';

  @override
  String get confirm => 'Confirm';

  @override
  String get confirm_deletion => 'Confirm Deletion';

  @override
  String get confirm_publish => 'Confirm publishing';

  @override
  String get confirm_share => 'Confirm share';

  @override
  String get connect => 'Connect';

  @override
  String get contribute => 'Hozzájárulás';

  @override
  String get couldnt_start_navigation =>
      'Couldn\'t start navigation. Check your connection and try again.';

  @override
  String get location_services_disabled =>
      'Location services are disabled. Please enable GPS to use navigation.';

  @override
  String get location_permission_denied =>
      'Location permission is required for navigation.';

  @override
  String get location_permission_permanently_denied =>
      'Location permission is permanently denied. Please enable it in Settings.';

  @override
  String get location_unavailable =>
      'Unable to determine your location. Please try again.';

  @override
  String get copy_link => 'Copy Link';

  @override
  String get create_new_list => 'Új lista létrehozása';

  @override
  String get create_waypoint => 'Create waypoint';

  @override
  String get creation_date => 'létrehozás dátuma';

  @override
  String get crop => 'Crop';

  @override
  String get cross => 'Cross';

  @override
  String get current_password => 'Current password';

  @override
  String get cycling => 'Cycling';

  @override
  String get cycling_speed => 'Cycling Speed';

  @override
  String get czech => 'Czech';

  @override
  String get danger_zone => 'Veszélyes zóna';

  @override
  String get date => 'Dátum';

  @override
  String get default_category => 'Default category';

  @override
  String get default_location => 'Alapértelmezett hely';

  @override
  String get degrees => 'Degrees';

  @override
  String get delete => 'Törlés';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Fiók törlése';

  @override
  String get delete_list_confirm =>
      'Tényleg törölni akarja ezt a listát? A listán szereplő nyomvonalak továbbra is elérhetőek maradnak.';

  @override
  String get delete_summit_log_confirm =>
      'Do you really want to delete this summit log? This action cannot be undone.';

  @override
  String get delete_trail_confirm =>
      'Tényleg törölni akarja ezt a nyomvonalat? Ezt a műveletet nem lehet visszavonni!';

  @override
  String get describe_your_trail => 'Nyomvonal leírása';

  @override
  String get description => 'Leírás';

  @override
  String get difficult => 'Nehéz';

  @override
  String get difficulty => 'Nehézség';

  @override
  String get directions => 'Irányok';

  @override
  String get display => 'Display';

  @override
  String get display_as => 'Megjelenítés mint';

  @override
  String get distance => 'Távolság';

  @override
  String get documentation => 'Dokumentáció';

  @override
  String get download => 'Download';

  @override
  String get draw_a_route => 'Draw a route';

  @override
  String get driving => 'Driving';

  @override
  String get duplicate => 'Duplicate';

  @override
  String get duration => 'Duration';

  @override
  String get dutch => 'Holland';

  @override
  String get easy => 'Könnyű';

  @override
  String get edit => 'Szerkesztés';

  @override
  String get edit_entry => 'Bejegyzés szerkesztése';

  @override
  String get edit_list => 'Lista szerkesztése';

  @override
  String get edit_route => 'Edit route';

  @override
  String get edit_waypoint => 'Útvonalpont szerkesztése';

  @override
  String get edited => 'edited';

  @override
  String get elevation_gain => 'Magasságnövekedés';

  @override
  String get elevation_loss => 'Elevation Loss';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => 'Email';

  @override
  String get email_not_unique => 'This email address is already in use.';

  @override
  String get email_updated => 'Email updated';

  @override
  String get email_verified => 'Email verified';

  @override
  String empty_activities(Object username) {
    return '$username has no activity yet';
  }

  @override
  String empty_bio(Object username) {
    return '$username has not added a bio yet';
  }

  @override
  String get empty_feed => 'Your feed is empty';

  @override
  String get empty_feed_explanation =>
      'Activities by you or people you follow will appear here';

  @override
  String empty_lists(Object username) {
    return '$username has no public lists';
  }

  @override
  String get enable_auto_routing => 'Enable auto-routing';

  @override
  String get english => 'Angol';

  @override
  String get entry => 'Bejegyzés';

  @override
  String get error_copying_trail => 'Error copying trail';

  @override
  String get error_creating_user => 'Hiba felhasználó hozzáadása közben';

  @override
  String get error_deleting_token => 'Error deleting token';

  @override
  String get error_disabling_strava_integration =>
      'Error disabling strava integration';

  @override
  String get error_during_login => 'Hiba bejelentkezés közben';

  @override
  String get error_during_password_reset =>
      'Unable to send password reset email';

  @override
  String get error_exporting_trail => 'Error exporting trail';

  @override
  String get error_generating_token => 'Error generating token';

  @override
  String get error_liking_trail => 'Error liking trail';

  @override
  String get error_logging_in_to_hammerhead => 'Error logging in to Hammerhead';

  @override
  String get error_logging_in_to_komoot => 'Error logging in to komoot';

  @override
  String get error_posting_comment => 'Error posting comment';

  @override
  String get error_printing_map => 'Error printing map';

  @override
  String get error_reading_file => 'Hiba a fájl olvasása közben';

  @override
  String get error_saving_list => 'Error saving list';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Error saving trail';

  @override
  String error_setting_up_integration(Object provider) {
    return 'Error setting up strava integration';
  }

  @override
  String get error_updating_hammerhead_integration =>
      'Error updating Hammerhead integration';

  @override
  String get error_updating_komoot_integration =>
      'Error updating komoot integration';

  @override
  String get error_updating_password => 'Error updating password';

  @override
  String get error_updating_strava_integration =>
      'Error updating komoot integration';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Error uploading trail to Hammerhead';

  @override
  String get est_duration => 'Becsült időtartam';

  @override
  String get everyone_with_the_link => 'Everyone with the link';

  @override
  String get expand_trail_list => 'Expand trail list';

  @override
  String get expiration => 'Expiration';

  @override
  String get expires => 'Expires';

  @override
  String get explore => 'Felfedezés';

  @override
  String get explore_some_trails => 'Fedezzen fel néhány ösvényt';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get stop_recording => 'Felvétel leállítása';

  @override
  String get stop_recording_confirm => 'Leállítja a felvételt?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get export => 'Export';

  @override
  String get export_all_trails => 'Export all trails';

  @override
  String get favourite_sport => 'Favourite sport';

  @override
  String get features => 'Jellemzők';

  @override
  String get ferry => 'Ferry';

  @override
  String get file_format => 'File format';

  @override
  String file_too_big(Object file, Object size) {
    return 'File $file is too big (max. $size)';
  }

  @override
  String get filter_categories => 'Filter categories';

  @override
  String get filter_difficulty => 'Filter difficulty';

  @override
  String get filter_tags => 'Filter tags';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => 'Finish';

  @override
  String get finish_disabled_hint =>
      'Add at least 2 anchors to finish your route.';

  @override
  String get fixed_speed => 'Fixed Speed';

  @override
  String get focus_map_on => 'Focus map on';

  @override
  String get follow => 'Follow';

  @override
  String get follow_request_pending => 'Request pending';

  @override
  String get followers => 'Followers';

  @override
  String get following => 'Following';

  @override
  String get food => 'Food';

  @override
  String get food_drinks => 'Food & Drinks';

  @override
  String get forgot_your_password => 'Forgot your password?';

  @override
  String get french => 'Francia';

  @override
  String get from_file => 'From file';

  @override
  String get from_photos => 'From Photos';

  @override
  String get from_url => 'From URL';

  @override
  String get garage => 'Garage';

  @override
  String get gas_station => 'Gas station';

  @override
  String get generate_new_token => 'Generate new token';

  @override
  String get german => 'Német';

  @override
  String get get_position_from_exif => 'Get coordinates from EXIF data';

  @override
  String get get_started => 'Get started';

  @override
  String get grid => 'Grid';

  @override
  String get grocery_store => 'Grocery store';

  @override
  String get hammerhead_integration_after_date_hint =>
      'If your hammerhead account is already synced with other trail databases, such as komoot or Strava, start syncing your Hammerhead data may result in duplicates. To avoid this, you can set an start date below, meaning only activities recorded after this date will be synced.';

  @override
  String get heading => 'Heading';

  @override
  String get height => 'Height';

  @override
  String get help => 'Help';

  @override
  String get hero_section_0_text =>
      'Explore exciting trails, save your favorites, and experience the beauty of nature. Find your next adventure!';

  @override
  String get hero_section_1_heading => 'It seems there are no trails here yet.';

  @override
  String get hero_section_1_text =>
      'Here are some trails you might like. Or you can just go to the full list right now.';

  @override
  String get hero_section_1_text_alternative =>
      'Get started by saving your latest adventure.';

  @override
  String get hero_section_2_text =>
      'Did you know? You cannot only save you hiking trails. There are many categories for all your adventures.';

  @override
  String get hiking => 'Hiking';

  @override
  String get home => 'Home';

  @override
  String get hotel => 'Hotel';

  @override
  String get hungarian => 'Magyar';

  @override
  String get hut => 'Hut';

  @override
  String get hybrid => 'Hybrid';

  @override
  String get icon => 'Ikon';

  @override
  String get ignore_trails_before_date => 'Ignore trails before this date';

  @override
  String get imperial => 'Angolszász';

  @override
  String get import => 'Import';

  @override
  String get import_hint => 'Select or drag GPX, FIT, KML or TCX files here...';

  @override
  String get include_description => 'Include description';

  @override
  String get include_waypoints => 'Útpontok hozzáadása';

  @override
  String get integration_description_hammerhead =>
      'Syncs your Hammerhead tours with wanderer in regular intervals.';

  @override
  String get integration_description_komoot =>
      'Syncs your komoot tours with wanderer in regular intervals.';

  @override
  String get integration_description_strava =>
      'Syncs your strava routes & activities with wanderer in regular intervals.';

  @override
  String get integration_disabled => 'integration disabled';

  @override
  String get integration_enabled => 'integration enabled';

  @override
  String get integration_privacy_hint_original =>
      'Imported trails will maintain the same visibility they have on the external platform. For example, if the original trail was public, it will be public in wanderer, even if trails are private by default according to your privacy settings.';

  @override
  String get integration_privacy_hint_user =>
      'The original trail\'s visibility is discarded. Instead, the local privacy settings for trails are applied to all imported trails.';

  @override
  String get integrations => 'Integrations';

  @override
  String get invalid_date => 'Érvénytelen dátum';

  @override
  String get invalid_username => 'Érvénytelen felhasználó';

  @override
  String get italian => 'Olasz';

  @override
  String get joined => 'Joined';

  @override
  String get keep_original => 'Keep original';

  @override
  String get keep_private => 'Keep private';

  @override
  String get language => 'Nyelf';

  @override
  String get last_used => 'Last used';

  @override
  String get latitude => 'Szélesség';

  @override
  String layer(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Layers',
      one: 'Layer',
    );
    return '$_temp0';
  }

  @override
  String get license => 'License';

  @override
  String get like_status => 'Like Status';

  @override
  String get liked => 'Liked';

  @override
  String get likes => 'Likes';

  @override
  String get limited => 'Limited';

  @override
  String get link_copied => 'Link copied!';

  @override
  String get linked_lists => 'Linked lists';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Listák',
      one: 'Lista',
    );
    return '$_temp0';
  }

  @override
  String get list_not_shared => 'Not shared with anyone';

  @override
  String get list_public_warning =>
      'All trails in this list will become public.';

  @override
  String get list_saved_successfully => 'List saved successfully';

  @override
  String get list_share_warning =>
      'Sharing a list automatically shares all trails contained in it.';

  @override
  String get list_share_warning_update =>
      'Added trails will be shared with everyone that has access to this list.';

  @override
  String get location => 'Helyszín';

  @override
  String get locations => 'Locations';

  @override
  String get location_tracking_notification_title => 'Wanderer';

  @override
  String get location_tracking_notification_text => 'Recording your trail';

  @override
  String get login => 'Bejelentkezés';

  @override
  String get login_details => 'Login details';

  @override
  String get logout => 'Kijelentkezés';

  @override
  String get longitude => 'Hosszúság';

  @override
  String get loop => 'Loop';

  @override
  String get make_one => 'Készítsen egyet!';

  @override
  String get make_thumbnail => 'Készítsen miniatűrképet';

  @override
  String get map => 'Térkép';

  @override
  String get map_style => 'Map style';

  @override
  String get max_hiking_difficulty => 'Max. Hiking Difficulty';

  @override
  String get metric => 'Metrikus';

  @override
  String get moderate => 'Mérsékelt';

  @override
  String get more => 'More';

  @override
  String get more_route_settings => 'More route settings';

  @override
  String get mountain => 'Mountain';

  @override
  String get mountain_pass => 'Mountain pass';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return 'Legalább $n karakter hosszúnak kell lennie';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return 'Must be at most $n characters long';
  }

  @override
  String get my_account => 'My Account';

  @override
  String get my_trails => 'My trails';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return '$n days ago';
  }

  @override
  String n_hours_ago(Object n) {
    return '$n hours ago';
  }

  @override
  String n_minutes_ago(Object n) {
    return '$n minutes ago';
  }

  @override
  String n_months_ago(Object n) {
    return '$n months ago';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n seconds ago',
      zero: 'just now',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return '$n years ago';
  }

  @override
  String get name => 'Név';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => 'Közelben';

  @override
  String get never => 'Never';

  @override
  String get new_list => 'Új lista';

  @override
  String get new_password => 'New password';

  @override
  String get new_password_error => 'Error setting new password';

  @override
  String get new_password_success => 'The new password has been set';

  @override
  String get new_password_text => 'Set a new password';

  @override
  String get new_token_generated => 'New API Token generated';

  @override
  String get new_trail => 'Új útvonal';

  @override
  String get trail_source_planner => 'Open trail planner';

  @override
  String get trail_source_record => 'Record trail';

  @override
  String get trail_source_import => 'Import file';

  @override
  String get trail_source_import_error => 'Could not import file';

  @override
  String get travel_profile_hike => 'Hike';

  @override
  String get travel_profile_hike_description =>
      'Plan a route for walking or hiking.';

  @override
  String get travel_profile_bike => 'Bike';

  @override
  String get travel_profile_bike_description => 'Plan a route for cycling.';

  @override
  String get coming_soon => 'Coming soon';

  @override
  String get no_account => 'Nincs még fiókja?';

  @override
  String get no_api_tokens => 'You have no API Tokens';

  @override
  String get no_comments_so_far => 'No comments so far';

  @override
  String get no_data => 'No data';

  @override
  String get no_description_for_now => 'No description for now';

  @override
  String get no_gps_data_in_image => 'No GPS data in image';

  @override
  String get no_grid => 'No Grid';

  @override
  String get no_notifications => 'No notifications';

  @override
  String get no_photos_here => 'No photos here';

  @override
  String get no_preference => 'Nincs preferált';

  @override
  String get no_results => 'Nincs eredmény';

  @override
  String get no_routes_added => 'No routes added';

  @override
  String get no_waypoints_yet => 'No waypoints yet';

  @override
  String get norwegian => 'Norwegian';

  @override
  String get not_a_valid_email_address => 'Érvénytelen e-mail cím';

  @override
  String get not_a_valid_url => 'Not a valid URL';

  @override
  String get not_completed => 'Nem teljesített';

  @override
  String notification_comment_mention(Object user) {
    return '$user mentioned you in a comment';
  }

  @override
  String notification_list_create(Object user) {
    return '$user created a new list';
  }

  @override
  String notification_list_share(Object user) {
    return '$user shared a list with you';
  }

  @override
  String get notification_new_follower => 'You have a new follower';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user created a summit log on your trail \"$trail\"';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user mentioned you in a summit log';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user left a comment on your trail \"$trail\"';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user created a new trail';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user liked your trail \"$trail\"';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user mentioned you in a trail';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user shared a trail with you';
  }

  @override
  String get notifications => 'Notifications';

  @override
  String object_share_error(Object object) {
    return 'A $object must be public to be shared across instances.';
  }

  @override
  String get off => 'Off';

  @override
  String get only_me => 'Only me';

  @override
  String get open_in_new_tab => 'Open in new tab';

  @override
  String get or => 'vagy';

  @override
  String get orientation => 'Orientation';

  @override
  String get paper_size => 'Paper size';

  @override
  String get paragraph => 'Paragraph';

  @override
  String get parking => 'Parking';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Jelszó';

  @override
  String get password_confirm => 'Confirm password';

  @override
  String get password_reset_sent => 'A password reset email has been sent';

  @override
  String get password_reset_text =>
      'We will send a reset link to your email address.';

  @override
  String get password_updated => 'Password updated';

  @override
  String get passwords_must_match => 'Passwords must match';

  @override
  String get photos => 'Képek';

  @override
  String photos_skipped_no_gps(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count photos skipped — no GPS data',
      one: '1 photo skipped — no GPS data',
    );
    return '$_temp0';
  }

  @override
  String get pick_a_trail => 'Válasszon útvonalat';

  @override
  String get planned_a_trail => 'planned a trail';

  @override
  String get planned_tours => 'Planned tours';

  @override
  String get pois => 'POIs';

  @override
  String get polish => 'Lengyel';

  @override
  String get portuguese => 'Portugál';

  @override
  String get print => 'Print';

  @override
  String get privacy => 'Privacy';

  @override
  String get private => 'Private';

  @override
  String get profile => 'Profil';

  @override
  String get public => 'Publikus';

  @override
  String get public_access => 'Public access';

  @override
  String get public_share_everyone =>
      'Everyone on the internet with the link can see this trail';

  @override
  String get public_share_limited =>
      'Only people with access can open the link';

  @override
  String get public_transport => 'Public transport';

  @override
  String get radius => 'Átmérő';

  @override
  String get railway_station => 'Railway station';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get read_more => 'Read more';

  @override
  String get ready_to_join => 'Ready to join';

  @override
  String get recalculate_elevation_data => 'Recalculate elevation data';

  @override
  String get recalculating_elevation_data_hint =>
      'Recalculating elevation data will erase the existing elevation data, if any, and replace it with data from Valhalla.';

  @override
  String get register => 'Regisztráció';

  @override
  String get remote_users_cannot_edit => 'Remote users cannot edit';

  @override
  String get removed_trail_from => 'Eltávolított nyomvonal a';

  @override
  String get removed_trails_from => 'Eltávolított nyomvonalak a';

  @override
  String get required => 'Kötelező';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Reset';

  @override
  String get reset_password => 'Reset Password';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => 'Folytatja a felvételt?';

  @override
  String get reverse_direction => 'Reverse direction';

  @override
  String get road => 'Road';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Routes',
      one: 'Route',
    );
    return '$_temp0';
  }

  @override
  String get route_point => 'Route Point';

  @override
  String get russian => 'Russian';

  @override
  String get save => 'Mentés';

  @override
  String get save_list => 'Save List';

  @override
  String get save_track => 'Save track';

  @override
  String get save_trail => 'Útvonal mentése';

  @override
  String get save_your_trail_first => 'Először mentsd el a nyomvonaladat';

  @override
  String get search => 'Search';

  @override
  String get search_cities => 'Városok keresése';

  @override
  String get search_for_trails_places => 'Nyomvonalak, helyek keresése';

  @override
  String get search_list => 'Search list';

  @override
  String get search_places => 'Search places';

  @override
  String get search_trails => 'Nyomvonalak keresése';

  @override
  String get select_date => 'Select date';

  @override
  String get select_list => 'Lista kiválasztása';

  @override
  String get selected => 'selected';

  @override
  String get send_to => 'Send to...';

  @override
  String get set_private => 'Set private';

  @override
  String get set_public => 'Set public';

  @override
  String get settings => 'Beállítások';

  @override
  String get settings_notification_comment_mention =>
      'Someone mentioned you in a comment';

  @override
  String get settings_notification_list_share =>
      'Someone shared a list with you';

  @override
  String get settings_notification_new_follower => 'You have a new follower';

  @override
  String get settings_notification_summit_log_create =>
      'Someone created a summit log on your trail';

  @override
  String get settings_notification_summit_log_mention =>
      'Someone mentioned you in a summit log';

  @override
  String get settings_notification_trail_comment =>
      'Someone left a comment on your trail';

  @override
  String get settings_notification_trail_like => 'Somone liked your trail';

  @override
  String get settings_notification_trail_mention =>
      'Someone mentioned you in a trail';

  @override
  String get settings_notification_trail_share =>
      'Someone shared a trail with you';

  @override
  String get settings_privacy_account_private =>
      'Only you can see your profile. You will not appear in search results. Other users cannot follow you or share trails with you. You can still publish trails or lists.';

  @override
  String get settings_privacy_account_public =>
      'Everyone can see your profile. You appear in search results. Other users can follow you and share trails with you.';

  @override
  String get settings_privacy_lists_private =>
      'Your lists are private by default. No one except you will be able to see them. You can change this setting at any point for individual lists.';

  @override
  String get settings_privacy_lists_public =>
      'Your lists are public by default. Everyone will be able to see them. You can change this setting at any point for individual lists.';

  @override
  String get settings_privacy_trails_private =>
      'Your trails are private by default. No one except you will be able to see them. You can change this setting at any point for individual trails.';

  @override
  String get settings_privacy_trails_public =>
      'Your trails are public by default. Everyone will be able to see them. You can change this setting at any point for individual trails.';

  @override
  String get settings_saved => 'Settings saved';

  @override
  String get share => 'Share';

  @override
  String get share_profile => 'Share profile';

  @override
  String get share_this_list => 'Share this list';

  @override
  String get share_this_trail => 'Share this trail';

  @override
  String get shared => 'Shared';

  @override
  String get shared_by => 'Shared by';

  @override
  String get shared_with => 'Shared with';

  @override
  String get shelter => 'Shelter';

  @override
  String get shortest => 'shortest';

  @override
  String get show_in_overview => 'Áttekintés';

  @override
  String get show_less => 'Show less';

  @override
  String get show_on_map => 'Mutatás térképen';

  @override
  String get shower => 'Shower';

  @override
  String get skiing => 'Skiing';

  @override
  String get slogan => 'Mentse el a kalandjait!';

  @override
  String get slope => 'Slope';

  @override
  String get someone => 'Someone';

  @override
  String get sort => 'Rendezés';

  @override
  String get spanish => 'spanyol';

  @override
  String get speed => 'Speed';

  @override
  String get start => 'Start';

  @override
  String get statistics => 'Statistics';

  @override
  String get stop_drawing => 'Stop drawing';

  @override
  String get stop_editing => 'Stop editing';

  @override
  String get strava_integration_after_date_hint =>
      'If your account has a large amount of acitivities you may run into Strava\'s API rate limit preventing you from syncing all activities at once. To mitigate this issue you can set an \"After\" date below so that only activities that were recorded after this date are synced.';

  @override
  String get subcategories => 'Alkategóriák';

  @override
  String get subway_stop => 'Subway entrance';

  @override
  String get summit => 'Summit';

  @override
  String get summit_book => 'Csúcspont könyv';

  @override
  String get table => 'Táblázat';

  @override
  String get tags => 'Tags';

  @override
  String get text => 'Szöveg';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get tilesets => 'Custom tilesets';

  @override
  String get time => 'Time';

  @override
  String get time_in_motion => 'Time in Motion';

  @override
  String get toilets => 'Toilets';

  @override
  String get top_speed => 'Top Speed';

  @override
  String get tourism => 'Tourism';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Útvonalak',
      one: 'Útvonal',
    );
    return '$_temp0';
  }

  @override
  String get trail_copied_successfully => 'trail copied successfully';

  @override
  String get trail_has_no_gpx => 'This trail has no GPX data.';

  @override
  String get trail_not_in_list => 'Trail is not in any list';

  @override
  String get trail_not_shared => 'Not shared with anyone';

  @override
  String get trail_saved_successfully => 'Trail saved successfully';

  @override
  String get some_waypoints_failed_to_save =>
      'Trail saved, but some waypoints failed to save';

  @override
  String get trails_for_you => 'Útvonalak önnek';

  @override
  String get tram_stop => 'Tram stop';

  @override
  String get unchanged => 'unchanged';

  @override
  String get units => 'Mértékegységek';

  @override
  String get unlink => 'Unlink';

  @override
  String get upload_file => 'Fájl feltöltése';

  @override
  String get upload_gpx => 'GPX feltöltése';

  @override
  String get upload_new_file => 'Upload new file';

  @override
  String get uploaded => 'uploaded';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Successfully uploaded trail to Hammerhead';

  @override
  String get use_hills => 'Use hills';

  @override
  String get use_roads => 'Use Roads';

  @override
  String get users => 'Users';

  @override
  String get username => 'Felhasználónév';

  @override
  String get username_not_unique =>
      'This username is already taken. Please try another.';

  @override
  String get view => 'View';

  @override
  String get viewpoint => 'Viewpoint';

  @override
  String get visibilty => 'Visibility';

  @override
  String get visibilty_status => 'Visibility status';

  @override
  String get walking_speed => 'Walking speed';

  @override
  String get water => 'Water';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Waypoints',
      one: 'Waypoint',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get width => 'Width';

  @override
  String get wrong_username_or_password =>
      'Helytelen felhasználónév vagy jelszó';

  @override
  String get you_can => 'You can';

  @override
  String get you_have_arrived => 'You\'ve arrived';

  @override
  String get settings_categories_confirm_disable_title => 'Hide this category?';

  @override
  String get settings_categories_confirm_disable_subcategory_title =>
      'Hide this subcategory?';

  @override
  String settings_categories_confirm_disable_body(int count) {
    return '$count of your trails use this category. They will stay published but this filter will be hidden.';
  }

  @override
  String get settings_categories_confirm_view_trails => 'View trails';

  @override
  String get settings_categories_confirm_disable_confirm => 'Disable anyway';

  @override
  String get settings_categories_empty_title => 'No subcategories';

  @override
  String get settings_categories_empty_body =>
      'This category has no subcategories to configure.';

  @override
  String get settings_categories_reorder_hint =>
      'Categories control which trail types you see and in what order. Turn one off to hide it as a filter — your trails stay published, they just won\'t appear under that category. Tap a category to manage its subcategories individually.\n\nTo change the order, press and hold a row, then drag it to a new position. The order you set here is reflected everywhere categories are shown.';
}
