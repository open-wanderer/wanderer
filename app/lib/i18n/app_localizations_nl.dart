// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Dutch Flemish (`nl`).
class AppLocalizationsNl extends AppLocalizations {
  AppLocalizationsNl([String locale = 'nl']) : super(locale);

  @override
  String get about => 'Over';

  @override
  String get appearance => 'Weergave';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Je staat op het punt je account te verwijderen. Al je routes worden hierdoor eveneens verwijderd. Wil je doorgaan?';

  @override
  String get account_privacy => 'Account privacy';

  @override
  String get add_bio => 'Voeg Bio toe';

  @override
  String get add_waypoint => 'Routepunt toevoegen';

  @override
  String get adjust_track => 'Pas spoor aan';

  @override
  String get after => 'Na';

  @override
  String get all => 'Alles';

  @override
  String get altitude => 'Hoogte';

  @override
  String get author => 'Auteur';

  @override
  String get average_speed => 'Gem. Snelheid';

  @override
  String get background_location_body =>
      'wanderer collects location data in the background so your trail keeps recording when the screen is off or the app is closed. Your recorded track stays on your device until you choose to save the trail.\n\nAndroid only offers this in system settings: open Location and choose \"Allow all the time\".';

  @override
  String get background_location_confirm => 'Open settings';

  @override
  String get background_location_title => 'Keep recording in the background';

  @override
  String get basic_info => 'Algemene informatie';

  @override
  String get before => 'Voor';

  @override
  String get by => 'door';

  @override
  String get cancel => 'Annuleren';

  @override
  String get discard => 'Verwijderen';

  @override
  String get discard_trail_confirm =>
      'Verwijder dit spoor en de bijbehorende wijzigingen?';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Kaarten',
      one: 'Kaart',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Categorieën';

  @override
  String get category => 'Categorie';

  @override
  String get change_email => 'E-mail wijzigen';

  @override
  String get change_password => 'Wachtwoord wijzigen';

  @override
  String get clear_all => 'Wis alles';

  @override
  String get close => 'Sluiten';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Opmerkingen',
      one: 'Opmerking',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Voltooid';

  @override
  String get completion_status => 'Voltooiingsstatus';

  @override
  String get confirm_deletion => 'Bevestig verwijderen';

  @override
  String get couldnt_start_navigation =>
      'Kon navigatie niet starten. Controleer je verbinding en probeer het opnieuw.';

  @override
  String get location_services_disabled =>
      'Locatieservices zijn uitgeschakeld. Activeer GPS om navigatie te gebruiken.';

  @override
  String get location_permission_denied =>
      'Locatie toestemming is vereist voor navigatie.';

  @override
  String get location_permission_permanently_denied =>
      'Locatie toestemming is permanent geweigerd. Activeer het in de Instellingen.';

  @override
  String get location_unavailable =>
      'Unable to determine your location. Please try again.';

  @override
  String get copy_link => 'Kopieer Link';

  @override
  String get create_waypoint => 'Nieuw routepunt';

  @override
  String get creation_date => 'Aanmaakdatum';

  @override
  String get current_password => 'Huidig wachtwoord';

  @override
  String get danger_zone => 'Gevarenzone';

  @override
  String get date => 'Datum';

  @override
  String get delete => 'Verwijderen';

  @override
  String get not_now => 'Not now';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Account verwijderen';

  @override
  String get delete_trail_confirm =>
      'Weet je zeker dat je deze route wilt verwijderen? Deze actie is onomkeerbaar.';

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
  String get description => 'Omschrijving';

  @override
  String get difficult => 'Moeilijk';

  @override
  String get difficulty => 'Moeilijkheidsgraad';

  @override
  String get directions => 'Routebeschrijving';

  @override
  String get distance => 'Afstand';

  @override
  String get download => 'Download';

  @override
  String get duration => 'Duur';

  @override
  String get easy => 'Makkelijk';

  @override
  String get edit => 'Bewerk';

  @override
  String get edit_needs_connection =>
      'Editing works on the server copy of this trail. Connect to the internet to edit it.';

  @override
  String get edit_waypoint => 'Bewerk Routepunt';

  @override
  String get edited => 'bewerkt';

  @override
  String get elevation_gain => 'Hoogteverschil';

  @override
  String get elevation_loss => 'Hoogteverlies';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => 'E-mail';

  @override
  String get email_not_unique => 'Dit e-mailadres is al in gebruik.';

  @override
  String get email_updated => 'Email bijgewerkt';

  @override
  String get error_deleting_trail => 'Error deleting trail';

  @override
  String get error_reading_file => 'Fout bij inlezen bestand';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Fout bij bewaren van route';

  @override
  String get error_updating_password => 'Fout bij bijwerken van wachtwoord';

  @override
  String get explore => 'Verkennen';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get stop_recording => 'Opname stoppen';

  @override
  String get stop_recording_confirm => 'Opname stoppen?';

  @override
  String get search_this_area => 'Search this area';

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
  String get follow => 'Volg';

  @override
  String get followers => 'Volgers';

  @override
  String get following => 'Volgend';

  @override
  String get from_photos => 'Van Foto\'s';

  @override
  String get help => 'Help';

  @override
  String get hiking => 'Hiking';

  @override
  String get home => 'Home';

  @override
  String get icon => 'Pictogram';

  @override
  String get imperial => 'Imperiaal';

  @override
  String get joined => 'Aangesloten';

  @override
  String get language => 'Taal';

  @override
  String get latitude => 'Breedtegraad';

  @override
  String get like_status => 'Like status';

  @override
  String get liked => 'Leuk gevonden';

  @override
  String get likes => 'Vind-ik-leuks';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Lijsten',
      one: 'Lijst',
    );
    return '$_temp0';
  }

  @override
  String get location => 'Locatie';

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
  String get login => 'Inloggen';

  @override
  String get logout => 'Uitloggen';

  @override
  String get longitude => 'Lengtegraad';

  @override
  String get map => 'Kaart';

  @override
  String get metric => 'Metrisch';

  @override
  String get moderate => 'Gemiddeld';

  @override
  String get my_account => 'Mijn Account';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String get name => 'Naam';

  @override
  String get navigate => 'Navigate';

  @override
  String get new_password => 'Nieuw wachtwoord';

  @override
  String get new_password_success => 'Het nieuwe wachtwoord is ingesteld.';

  @override
  String get new_trail => 'Nieuwe Route';

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
  String get no_comments_so_far => 'Tot nu toe geen opmerkingen';

  @override
  String get no_data => 'Geen data';

  @override
  String get no_description_for_now => 'Voorlopig geen beschrijving';

  @override
  String get no_gps_data_in_image => 'Geen GPS data in afbeelding';

  @override
  String get no_preference => 'Geen voorkeur';

  @override
  String get no_trails_found => 'No trails found';

  @override
  String get not_completed => 'Niet voltooid';

  @override
  String get notifications => 'Meldingen';

  @override
  String get only_me => 'Alleen ik';

  @override
  String get or => 'of';

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
  String get password => 'Wachtwoord';

  @override
  String get password_confirm => 'Bevestig wachtwoord';

  @override
  String get passwords_must_match => 'Wachtwoorden moeten overeenkomen';

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
  String get photos => 'Foto\'s';

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
  String get privacy => 'Privacy';

  @override
  String get private => 'Privaat';

  @override
  String get profile => 'Profiel';

  @override
  String get public => 'Openbaar';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get register => 'Registreren';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Herstellen';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => 'Opname hervatten?';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Tochten',
      one: 'Tocht',
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
  String get save => 'Bewaren';

  @override
  String get save_recording_options => 'Save recording';

  @override
  String get save_track => 'Save track';

  @override
  String get search => 'Search';

  @override
  String get search_for_trails_places =>
      'Zoek naar routes, lijsten en locaties';

  @override
  String get select_date => 'Select date';

  @override
  String get settings => 'Instellingen';

  @override
  String get settings_notification_comment_mention =>
      'Iemand vermeldt je in een reactie';

  @override
  String get settings_notification_list_share =>
      'Iemand heeft een lijst met je gedeeld';

  @override
  String get settings_notification_new_follower => 'Je hebt een nieuwe volger';

  @override
  String get settings_notification_summit_log_create =>
      'Iemand heeft een topboek van je trail gemaakt';

  @override
  String get settings_notification_summit_log_mention =>
      'Iemand vermeldde je in een topboek';

  @override
  String get settings_notification_trail_comment =>
      'Iemand heeft een opmerking geplaatst bij je route';

  @override
  String get settings_notification_trail_like => 'Iemand vindt je route leuk';

  @override
  String get settings_notification_trail_mention =>
      'Iemand heeft je in een route genoemd';

  @override
  String get settings_notification_trail_share =>
      'Iemand heeft een route met je gedeeld';

  @override
  String get settings_privacy_account_private =>
      'Alleen jij kunt je profiel zien. Je verschijnt niet in de zoekresultaten. Andere gebruikers kunnen je niet volgen of routes met je delen. Je kunt nog steeds routes of lijsten publiceren.';

  @override
  String get settings_privacy_account_public =>
      'Iedereen kan je profiel zien. Je verschijnt in de zoekresultaten. Andere gebruikers kunnen je volgen en ervaringen met je delen.';

  @override
  String get settings_privacy_lists_private =>
      'Je lijsten zijn standaard privé. Niemand behalve jij kan ze zien. Je kunt deze instelling op elk moment voor individuele lijsten wijzigen.';

  @override
  String get settings_privacy_lists_public =>
      'Je lijsten zijn standaard openbaar. Iedereen kan ze zien. Je kunt deze instelling op elk moment voor individuele lijsten wijzigen.';

  @override
  String get settings_privacy_trails_private =>
      'Je routes zijn standaard privé. Niemand behalve jij kan ze zien. Je kunt deze instelling op elk moment voor individuele routes wijzigen.';

  @override
  String get settings_privacy_trails_public =>
      'Je routes zijn standaard openbaar. Iedereen kan ze zien. Je kunt deze instelling op elk moment voor individuele routes wijzigen.';

  @override
  String get share => 'Deel';

  @override
  String get share_profile => 'Deel profiel';

  @override
  String get shared => 'Gedeeld';

  @override
  String get show_on_map => 'Tonen op kaart';

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
  String get slogan => 'Jouw routes. Jouw data. Jouw server.';

  @override
  String get sort => 'Sorteren';

  @override
  String get speed => 'Snelheid';

  @override
  String get start => 'Start';

  @override
  String get subcategories => 'Subcategorieën';

  @override
  String get summit_book => 'Bergtopboek';

  @override
  String get sync_failed => 'Upload failed · Tap to retry';

  @override
  String get sync_pending => 'Waiting to upload';

  @override
  String get sync_uploading => 'Uploading…';

  @override
  String get tags => 'Labels';

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
      other: 'Routes',
      one: 'Route',
    );
    return '$_temp0';
  }

  @override
  String get trail_saved_successfully => 'Route succesvol bewaard';

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
  String get units => 'Eenheden';

  @override
  String get users => 'Users';

  @override
  String get username => 'Gebruikersnaam';

  @override
  String get username_not_unique =>
      'Deze gebruikersnaam is al in gebruik. Probeer een andere naam.';

  @override
  String get visibilty_status => 'Zichtbaarheid status';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Routepunten',
      one: 'Routepunt',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get wrong_username_or_password =>
      'Onjuiste gebruikersnaam of wachtwoord';

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
  String get open_in_new_tab => 'Open in een nieuwe tabblad';

  @override
  String get remove => 'Remove';

  @override
  String get remove_download_confirm_body =>
      'This removes the downloaded copy from this device. The trail itself is not deleted — you\'ll need to download it again to use it offline.';

  @override
  String get apply => 'Toepassen';

  @override
  String get add_at_least_2_anchors_hint =>
      'Add at least 2 anchors to see the elevation profile.';

  @override
  String get reverse_direction => 'Omgekeerde richting';

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
  String get edit_route => 'Bewerk route';

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
