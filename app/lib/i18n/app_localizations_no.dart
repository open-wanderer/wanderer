// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Norwegian (`no`).
class AppLocalizationsNo extends AppLocalizations {
  AppLocalizationsNo([String locale = 'no']) : super(locale);

  @override
  String get about => 'Om';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Du er i ferd med å slette kontoen din. Alle sporene dine vil også bli slettet. Ønsker du å fortsette?';

  @override
  String get account_privacy => 'Personvern for konto';

  @override
  String get add_bio => 'Legg til biografi';

  @override
  String get add_waypoint => 'Legg til veipunkt';

  @override
  String get adjust_track => 'Adjust track';

  @override
  String get after => 'Etter';

  @override
  String get all => 'All';

  @override
  String get altitude => 'Høyde';

  @override
  String get author => 'Forfatter';

  @override
  String get average_speed => 'Gj.snittsfart';

  @override
  String get basic_info => 'Grunnleggende info';

  @override
  String get before => 'Før';

  @override
  String get by => 'av';

  @override
  String get cancel => 'Avbryt';

  @override
  String get discard => 'Discard';

  @override
  String get discard_trail_confirm => 'Discard this trail and its changes?';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Kort',
      one: 'Kort',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Kategorier';

  @override
  String get category => 'Kategori';

  @override
  String get change_email => 'Endre e-post';

  @override
  String get change_password => 'Endre passord';

  @override
  String get clear_all => 'Fjern alle';

  @override
  String get close => 'Lukk';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Kommentarer',
      one: 'Kommentar',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Fullført';

  @override
  String get completion_status => 'Fullføringsstatus';

  @override
  String get confirm_deletion => 'Bekreft sletting';

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
  String get copy_link => 'Kopier lenke';

  @override
  String get create_waypoint => 'Opprett veipunkt';

  @override
  String get creation_date => 'Opprettelsesdato';

  @override
  String get current_password => 'Nåværende passord';

  @override
  String get danger_zone => 'Faresone';

  @override
  String get date => 'Dato';

  @override
  String get delete => 'Slett';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Slett konto';

  @override
  String get delete_trail_confirm =>
      'Vil du virkelig slette denne stien? Denne handlingen kan ikke angres.';

  @override
  String get delete_blocked_while_uploading =>
      'This trail is uploading right now. Wait for the upload to finish, then try again.';

  @override
  String get delete_unsynced_trail_confirm =>
      'Delete this trail? It hasn\'t been uploaded yet, so this can\'t be undone.';

  @override
  String get delete_needs_connection =>
      'This trail is already on the server. Connect to the internet to delete it.';

  @override
  String get description => 'Beskrivelse';

  @override
  String get difficult => 'Vanskelig';

  @override
  String get difficulty => 'Vanskelighetsgrad';

  @override
  String get directions => 'Veibeskrivelse';

  @override
  String get distance => 'Distanse';

  @override
  String get download => 'Last ned';

  @override
  String get duration => 'Varighet';

  @override
  String get easy => 'Lett';

  @override
  String get edit => 'Rediger';

  @override
  String get edit_needs_connection =>
      'Editing works on the server copy of this trail. Connect to the internet to edit it.';

  @override
  String get edit_waypoint => 'Rediger veipunkt';

  @override
  String get edited => 'redigert';

  @override
  String get elevation_gain => 'Høydemeter opp';

  @override
  String get elevation_loss => 'Høydemeter ned';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => 'E-post';

  @override
  String get email_not_unique => 'Denne e-postadressen er allerede i bruk.';

  @override
  String get email_updated => 'E-post oppdatert';

  @override
  String get error_deleting_trail => 'Error deleting trail';

  @override
  String get error_reading_file => 'Feil ved lesing av fil';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Feil ved lagring av sti';

  @override
  String get error_updating_password => 'Feil ved oppdatering av passord';

  @override
  String get explore => 'Utforsk';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get stop_recording => 'Stopp opptak';

  @override
  String get stop_recording_confirm => 'Stoppe opptaket?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get filter_tags => 'Filtrer tagger';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => 'Fullfør';

  @override
  String get finish_disabled_hint =>
      'Add at least 2 anchors to finish your route.';

  @override
  String get follow => 'Følg';

  @override
  String get followers => 'Følgere';

  @override
  String get following => 'Følger';

  @override
  String get from_photos => 'Fra bilder';

  @override
  String get help => 'Hjelp';

  @override
  String get hiking => 'Vandring';

  @override
  String get home => 'Hjem';

  @override
  String get icon => 'Ikon';

  @override
  String get imperial => 'Imperisk';

  @override
  String get joined => 'Ble medlem';

  @override
  String get language => 'Språk';

  @override
  String get latitude => 'Breddegrad';

  @override
  String get like_status => 'Liker-status';

  @override
  String get liked => 'Likte';

  @override
  String get likes => 'Likerklikk';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Lister',
      one: 'Liste',
    );
    return '$_temp0';
  }

  @override
  String get location => 'Plassering';

  @override
  String get locations => 'Locations';

  @override
  String get center_on_my_location => 'Center on my location';

  @override
  String get location_tracking_notification_title => 'wanderer';

  @override
  String get location_tracking_notification_text => 'Recording your trail';

  @override
  String location_tracking_notification_text_navigating(String trail) {
    return 'Navigating $trail';
  }

  @override
  String get login => 'Logg inn';

  @override
  String get logout => 'Logg ut';

  @override
  String get longitude => 'Lengdegrad';

  @override
  String get map => 'Kart';

  @override
  String get metric => 'Metrisk';

  @override
  String get moderate => 'Moderat';

  @override
  String get my_account => 'Min konto';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String get name => 'Navn';

  @override
  String get navigate => 'Navigate';

  @override
  String get new_password => 'Nytt passord';

  @override
  String get new_password_success => 'Det nye passordet er satt';

  @override
  String get new_trail => 'Ny sti';

  @override
  String get trail_source_planner => 'Open trail planner';

  @override
  String get trail_source_planner_description =>
      'Draw a new route on the map, waypoint by waypoint.';

  @override
  String get trail_source_record => 'Record trail';

  @override
  String get trail_source_record_description =>
      'Track your live coordinates and log your journey in real-time.';

  @override
  String get trail_source_import => 'Import file';

  @override
  String get trail_source_import_description =>
      'Upload GPX, KML, KMZ, TCX or FIT files directly from your device storage.';

  @override
  String get trail_source_import_error => 'Could not import file';

  @override
  String get trail_source_offline_import_error =>
      'Only GPX files can be imported offline';

  @override
  String get no_comments_so_far => 'Ingen kommentarer ennå';

  @override
  String get no_data => 'Ingen data';

  @override
  String get no_description_for_now => 'Ingen beskrivelse ennå';

  @override
  String get no_gps_data_in_image => 'Ingen GPS-data i bildet';

  @override
  String get no_preference => 'Ingen preferanse';

  @override
  String get no_trails_found => 'No trails found';

  @override
  String get not_completed => 'Ikke fullført';

  @override
  String get notifications => 'Varsler';

  @override
  String get only_me => 'Kun meg';

  @override
  String get or => 'eller';

  @override
  String get own_trails_empty_body =>
      'Trails you record or save offline appear here, and upload automatically once you\'re back online.';

  @override
  String get own_trails_empty_title => 'Nothing saved yet';

  @override
  String get own_trails_offline_banner =>
      'Offline — showing only trails on this device.';

  @override
  String get trails_on_device => 'Trails (on device)';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Passord';

  @override
  String get password_confirm => 'Bekreft passord';

  @override
  String get passwords_must_match => 'Passordene må være like';

  @override
  String photo_copy_failed_toast(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Trail saved, but $count photos couldn\'t be saved.',
      one: 'Trail saved, but 1 photo couldn\'t be saved.',
    );
    return '$_temp0';
  }

  @override
  String get photos => 'Bilder og videoer';

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
  String get privacy => 'Personvern';

  @override
  String get private => 'Privat';

  @override
  String get profile => 'Profil';

  @override
  String get public => 'Offentlig';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get register => 'Registrer';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Tilbakestill';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => 'Gjenoppta opptaket?';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Ruter',
      one: 'Rute',
    );
    return '$_temp0';
  }

  @override
  String get follow_roads => 'Follow roads';

  @override
  String get follow_roads_description =>
      'Snap the recorded path to the nearest roads and trails.';

  @override
  String get recalculate_heights => 'Recalculate heights';

  @override
  String get recalculate_heights_description =>
      'Replace recorded GPS elevation with more accurate values from the map.';

  @override
  String get save => 'Lagre';

  @override
  String get save_recording_options => 'Save recording';

  @override
  String get save_track => 'Save track';

  @override
  String get search => 'Search';

  @override
  String get search_for_trails_places => 'Søk etter stier, lister, steder';

  @override
  String get select_date => 'Select date';

  @override
  String get settings => 'Innstillinger';

  @override
  String get settings_notification_comment_mention =>
      'Noen nevnte deg i en kommentar';

  @override
  String get settings_notification_list_share => 'Noen delte en liste med deg';

  @override
  String get settings_notification_new_follower => 'Du har en ny følger';

  @override
  String get settings_notification_summit_log_create =>
      'Noen opprettet en toppbokelement på din sti';

  @override
  String get settings_notification_summit_log_mention =>
      'Noen nevnte deg i en toppbokelement';

  @override
  String get settings_notification_trail_comment =>
      'Noen la igjen en kommentar på din sti';

  @override
  String get settings_notification_trail_like => 'Noen likte din sti';

  @override
  String get settings_notification_trail_mention => 'Noen nevnte deg i en sti';

  @override
  String get settings_notification_trail_share => 'Noen delte en sti med deg';

  @override
  String get settings_privacy_account_private =>
      'Kun du kan se profilen din. Du vil ikke vises i søkeresultater. Andre brukere kan ikke følge deg eller dele stier med deg. Du kan fortsatt publisere stier eller lister.';

  @override
  String get settings_privacy_account_public =>
      'Alle kan se profilen din. Du vises i søkeresultater. Andre brukere kan følge deg og dele stier med deg.';

  @override
  String get settings_privacy_lists_private =>
      'Listene dine er private som standard. Ingen unntatt deg vil kunne se dem. Du kan endre denne innstillingen når som helst for individuelle lister.';

  @override
  String get settings_privacy_lists_public =>
      'Listene dine er offentlige som standard. Alle vil kunne se dem. Du kan endre denne innstillingen når som helst for individuelle lister.';

  @override
  String get settings_privacy_trails_private =>
      'Stiene dine er private som standard. Ingen unntatt deg vil kunne se dem. Du kan endre denne innstillingen når som helst for individuelle stier.';

  @override
  String get settings_privacy_trails_public =>
      'Stiene dine er offentlige som standard. Alle vil kunne se dem. Du kan endre denne innstillingen når som helst for individuelle stier.';

  @override
  String get share => 'Del';

  @override
  String get share_profile => 'Del profil';

  @override
  String get shared => 'Delt';

  @override
  String get show_on_map => 'Vis på kart';

  @override
  String signout_unsynced_warning(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'You have $count trails not uploaded yet. Signing out won\'t delete them — they\'ll be here when you sign back in — but they won\'t upload until then.',
      one:
          'You have 1 trail not uploaded yet. Signing out won\'t delete it — it\'ll be here when you sign back in — but it won\'t upload until then.',
    );
    return '$_temp0';
  }

  @override
  String get slogan => 'Dine stier. Dine data. Din server.';

  @override
  String get sort => 'Sorter';

  @override
  String get speed => 'Hastighet';

  @override
  String get start => 'Start';

  @override
  String get subcategories => 'Underkategorier';

  @override
  String get summit_book => 'Toppbok';

  @override
  String get sync_failed => 'Upload failed · Tap to retry';

  @override
  String get sync_pending => 'Waiting to upload';

  @override
  String get sync_uploading => 'Uploading…';

  @override
  String get tags => 'Tagger';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get time_in_motion => 'Time in Motion';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Stier',
      one: 'Sti',
    );
    return '$_temp0';
  }

  @override
  String get trail_saved_successfully => 'Sti lagret';

  @override
  String get trail_not_on_this_device =>
      'This trail is no longer on this device.';

  @override
  String get trail_uploaded_reopen_to_edit =>
      'This trail finished uploading. Re-open it from your trails to keep editing.';

  @override
  String get some_waypoints_failed_to_save =>
      'Trail saved, but some waypoints failed to save';

  @override
  String get units => 'Enheter';

  @override
  String get users => 'Users';

  @override
  String get username => 'Brukernavn';

  @override
  String get username_not_unique =>
      'Dette brukernavnet er allerede tatt. Vennligst prøv et annet.';

  @override
  String get visibilty_status => 'Synlighetsstatus';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Veipunkter',
      one: 'Veipunkt',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get wrong_username_or_password => 'Feil brukernavn eller passord';

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

  @override
  String get something_went_wrong => 'Something went wrong';

  @override
  String get technical_details => 'Technical Details';

  @override
  String get link => 'Link';

  @override
  String get url => 'URL';

  @override
  String get open_in_new_tab => 'Åpne i ny fane';

  @override
  String get remove => 'Remove';

  @override
  String get remove_download_confirm_body =>
      'This removes the downloaded copy from this device. The trail itself is not deleted — you\'ll need to download it again to use it offline.';

  @override
  String get apply => 'Apply';

  @override
  String get add_at_least_2_anchors_hint =>
      'Add at least 2 anchors to see the elevation profile.';

  @override
  String get reverse_direction => 'Snu retning';

  @override
  String get delete_all => 'Delete all';

  @override
  String get auto_routing => 'Auto-routing';

  @override
  String get auto_routing_hint =>
      'Automatically follow roads and paths between anchors.';

  @override
  String get travel_profile => 'Travel profile';

  @override
  String get no_track_data => 'No track data';

  @override
  String get available_offline => 'Available offline';

  @override
  String get no_lists_found => 'No lists found';

  @override
  String get search_lists => 'Search lists…';

  @override
  String get search_for_a_location => 'Search for a location';

  @override
  String no_results_for_query(String query) {
    return 'No results for \"$query\"';
  }

  @override
  String get filter => 'Filter';

  @override
  String no_label_yet(String label) {
    return 'No $label yet.';
  }

  @override
  String get no_lists_yet => 'No lists yet.';

  @override
  String get no_bio_yet => 'No bio yet.';

  @override
  String get show_more => 'Show more';

  @override
  String get show_less => 'Show less';

  @override
  String get feed => 'Feed';

  @override
  String get no_trails_yet => 'No trails yet.';

  @override
  String get library_empty_title => 'No downloaded trails';

  @override
  String get library_empty_body =>
      'Trails you download are kept here so you can open them offline.';

  @override
  String get library_empty_search_body =>
      'Try a different search term or clear your filters.';

  @override
  String get search_location => 'Search location';

  @override
  String no_servers_match_query(String query) {
    return 'No servers match \"$query\"';
  }

  @override
  String get use_custom_url_instead => 'Use custom URL instead';

  @override
  String get select_instance => 'Select Instance';

  @override
  String get enter_server_url_hint => 'Enter server URL (e.g. wanderer.to)';

  @override
  String get search_library => 'Search library…';

  @override
  String language_and_units(String language, String units) {
    return '$language & $units';
  }

  @override
  String get edit_route => 'Rediger rute';

  @override
  String get undo => 'Undo';

  @override
  String get redo => 'Redo';

  @override
  String get bold => 'Bold';

  @override
  String get italic => 'Italic';

  @override
  String get underline => 'Underline';

  @override
  String get bullet_list => 'Bullet list';

  @override
  String get ordered_list => 'Ordered list';

  @override
  String get blockquote => 'Blockquote';

  @override
  String get library => 'Library';

  @override
  String get settings_offline_regions_title => 'Offline Maps/Regions';

  @override
  String get regions_search_hint => 'Search regions';

  @override
  String get regions_dem_toggle_label => 'Download elevation data (DEM)';

  @override
  String get regions_dem_toggle_caption =>
      'Adds hillshading; increases download size';

  @override
  String get regions_update_available => 'Update available';

  @override
  String get regions_update_action => 'Update';

  @override
  String get regions_retry => 'Retry';

  @override
  String get regions_not_yet_available => 'Not yet available';

  @override
  String get regions_build_failed => 'Build failed';

  @override
  String regions_delete_confirm_title(String name) {
    return 'Delete $name?';
  }

  @override
  String get regions_delete_confirm_body =>
      'This removes the downloaded map and elevation data for this region. You\'ll need to download it again to use it offline.';

  @override
  String get regions_delete_confirm_action => 'Delete';

  @override
  String regions_disk_usage_summary(String size, num count) {
    return '$size used across $count downloaded region(s)';
  }

  @override
  String get regions_empty_search_title => 'No matching regions';

  @override
  String get regions_empty_search_body => 'Try a different search term.';

  @override
  String get regions_empty_catalog_title => 'No offline regions available';

  @override
  String get regions_empty_catalog_body =>
      'Ask your Wanderer instance administrator to configure downloadable regions.';

  @override
  String get regions_vector_tile_title => 'Vector';

  @override
  String get regions_dem_tile_title => 'Elevation data';

  @override
  String get regions_download_failed => 'Download failed';

  @override
  String get regions_dem_locked_subtitle => 'Download map data first';

  @override
  String get regions_offline_unavailable_title => 'Can\'t load regions';

  @override
  String get regions_offline_unavailable_body =>
      'Connect to the internet to browse and manage downloadable regions.';

  @override
  String get regions_map_geometry_failed => 'Could not load region outline';

  @override
  String get regions_map_back_label => 'Back to regions';

  @override
  String regions_group_expand_label(String name) {
    return 'Expand $name';
  }

  @override
  String regions_group_collapse_label(String name) {
    return 'Collapse $name';
  }

  @override
  String get offline_title => 'You\'re offline';

  @override
  String get offline_try_again => 'Try again';

  @override
  String get offline_map_body =>
      'Connect to the internet to load the map. Downloaded trails are still available.';

  @override
  String get offline_list_body => 'Connect to the internet to load lists.';

  @override
  String get offline_profile_body =>
      'Connect to the internet to load your full profile.';

  @override
  String get offline_settings_banner =>
      'You\'re offline. Settings are read-only until you reconnect.';

  @override
  String get offline_action_unavailable =>
      'You\'re offline — try again once you\'re back online.';

  @override
  String get offline_categories_body =>
      'Connect to the internet to manage categories.';

  @override
  String get offline_trail_search_body =>
      'Connect to the internet to search for trails. Downloaded trails are still available.';
}
