// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Czech (`cs`).
class AppLocalizationsCs extends AppLocalizations {
  AppLocalizationsCs([String locale = 'cs']) : super(locale);

  @override
  String get about => 'O aplikaci';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Chystáte se odstranit svůj účet. Všechny vaše záznamy budou odstraněny též. Chcete pokračovat?';

  @override
  String get account_privacy => 'Ochrana osobních údajů';

  @override
  String get add_bio => 'Přidat informace o mně';

  @override
  String get add_waypoint => 'Přidat bod trasy';

  @override
  String get adjust_track => 'Adjust track';

  @override
  String get after => 'Po';

  @override
  String get all => 'All';

  @override
  String get altitude => 'Nadmořská výška';

  @override
  String get author => 'Autor';

  @override
  String get average_speed => 'Průměrná rychlost';

  @override
  String get basic_info => 'Základní informace';

  @override
  String get before => 'Před';

  @override
  String get by => 'od';

  @override
  String get cancel => 'Zrušit';

  @override
  String get discard => 'Discard';

  @override
  String get discard_trail_confirm => 'Discard this trail and its changes?';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Karet',
      few: 'Karty',
      one: 'Karta',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Kategorie';

  @override
  String get category => 'Kategorie';

  @override
  String get change_email => 'Změnit e-mail';

  @override
  String get change_password => 'Změnit heslo';

  @override
  String get clear_all => 'Vymazat vše';

  @override
  String get close => 'Zavřít';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Komentářů',
      few: 'Komentáře',
      one: 'Komentář',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Dokončeno';

  @override
  String get completion_status => 'Stav dokončení';

  @override
  String get confirm_deletion => 'Potvrdit odstranění';

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
  String get copy_link => 'Kopírovat odkaz';

  @override
  String get create_waypoint => 'Vytvořit bod trasy';

  @override
  String get creation_date => 'Datum vytvoření';

  @override
  String get current_password => 'Současné heslo';

  @override
  String get danger_zone => 'Nebezpečná zóna';

  @override
  String get date => 'Datum';

  @override
  String get delete => 'Vymazat';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Odstranit účet';

  @override
  String get delete_trail_confirm =>
      'Doopravdy chcete tuto trasu smazat? Tuto akci nelze vrátit zpět.';

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
  String get description => 'Popis';

  @override
  String get difficult => 'Náročné';

  @override
  String get difficulty => 'Obtížnost';

  @override
  String get directions => 'Navigovat';

  @override
  String get distance => 'Vzdálenost';

  @override
  String get download => 'Stáhnout';

  @override
  String get duration => 'Doba trvání';

  @override
  String get easy => 'Jednoduchý';

  @override
  String get edit => 'Upravit';

  @override
  String get edit_needs_connection =>
      'Editing works on the server copy of this trail. Connect to the internet to edit it.';

  @override
  String get edit_waypoint => 'Upravit bod trasy';

  @override
  String get edited => 'upraveno';

  @override
  String get elevation_gain => 'Stoupání';

  @override
  String get elevation_loss => 'Klesání';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => 'E-mail';

  @override
  String get email_not_unique => 'Tato e-mailová adresa je již obsazena.';

  @override
  String get email_updated => 'E-mail byl aktualizován';

  @override
  String get error_deleting_trail => 'Error deleting trail';

  @override
  String get error_reading_file => 'Soubor nelze načíst';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Chyba při ukládání trasy';

  @override
  String get error_updating_password => 'Chyba během aktualizace hesla';

  @override
  String get explore => 'Objevování';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get stop_recording => 'Zastavit nahrávání';

  @override
  String get stop_recording_confirm => 'Zastavit nahrávání?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get filter_tags => 'Filtrovat podle štítků';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => 'Konec';

  @override
  String get finish_disabled_hint =>
      'Add at least 2 anchors to finish your route.';

  @override
  String get follow => 'Sledovat';

  @override
  String get followers => 'Sledující';

  @override
  String get following => 'Sleduji';

  @override
  String get from_photos => 'Z fotografií';

  @override
  String get help => 'Nápověda';

  @override
  String get hiking => 'Turistika';

  @override
  String get home => 'Hlavní stránka';

  @override
  String get icon => 'Ikonka';

  @override
  String get imperial => 'Imperiální jednotky';

  @override
  String get joined => 'Připojeno';

  @override
  String get language => 'Jazyk';

  @override
  String get latitude => 'Zeměpisná šířka';

  @override
  String get like_status => 'Status „Líbí se mi“';

  @override
  String get liked => 'Líbilo se';

  @override
  String get likes => 'Líbí se';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Seznamů',
      few: 'Seznamy',
      one: 'Seznam',
    );
    return '$_temp0';
  }

  @override
  String get location => 'Poloha';

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
  String get login => 'Přihlásit se';

  @override
  String get logout => 'Odhlásit se';

  @override
  String get longitude => 'Zeměpisná délka';

  @override
  String get map => 'Mapa';

  @override
  String get metric => 'Metrický';

  @override
  String get moderate => 'Středně náročné';

  @override
  String get my_account => 'Můj účet';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String get name => 'Název';

  @override
  String get navigate => 'Navigate';

  @override
  String get new_password => 'Nové heslo';

  @override
  String get new_password_success => 'Nové heslo bylo nastaveno';

  @override
  String get new_trail => 'Nová trasa';

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
  String get no_comments_so_far => 'Zatím žádné komentáře';

  @override
  String get no_data => 'Žádná data';

  @override
  String get no_description_for_now => 'Zatím bez popisu';

  @override
  String get no_gps_data_in_image => 'Obrázek neobsahuje GPS data';

  @override
  String get no_preference => 'Bez preference';

  @override
  String get no_trails_found => 'No trails found';

  @override
  String get not_completed => 'Nedokončeno';

  @override
  String get notifications => 'Oznámení';

  @override
  String get only_me => 'Jen já';

  @override
  String get or => 'nebo';

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
  String get password => 'Heslo';

  @override
  String get password_confirm => 'Potvrďte heslo';

  @override
  String get passwords_must_match => 'Hesla se musí shodovat';

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
  String get photos => 'Fotografie a videa';

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
  String get privacy => 'Soukromí';

  @override
  String get private => 'Soukromé';

  @override
  String get profile => 'Profil';

  @override
  String get public => 'Veřejné';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get register => 'Registrace';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Obnovit';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => 'Obnovit nahrávání?';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Tras',
      few: 'Trasy',
      one: 'Trasa',
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
  String get save => 'Uložit';

  @override
  String get save_recording_options => 'Save recording';

  @override
  String get save_track => 'Save track';

  @override
  String get search => 'Search';

  @override
  String get search_for_trails_places => 'Vyhledat trasy, seznamy, místa';

  @override
  String get select_date => 'Select date';

  @override
  String get settings => 'Nastavení';

  @override
  String get settings_notification_comment_mention =>
      'Někdo vás zmínil v komentáři';

  @override
  String get settings_notification_list_share => 'Někdo s vámi sdílel seznam';

  @override
  String get settings_notification_new_follower => 'Máte nového sledujícího';

  @override
  String get settings_notification_summit_log_create =>
      'Někdo přidal záznam o výstupu na vaší trase';

  @override
  String get settings_notification_summit_log_mention =>
      'Někdo vás zmínil v záznamu o výstupu';

  @override
  String get settings_notification_trail_comment =>
      'Někdo zanechal komentář k vaší trase';

  @override
  String get settings_notification_trail_like => 'Někomu se líbí vaše trasa';

  @override
  String get settings_notification_trail_mention => 'Někdo vás zmínil u trasy';

  @override
  String get settings_notification_trail_share => 'Někdo s vámi sdílel trasu';

  @override
  String get settings_privacy_account_private =>
      'Váš profil vidíte pouze vy. Nebudete se zobrazovat ve výsledcích vyhledávání. Ostatní uživatelé vás nemohou sledovat ani s vámi sdílet trasy. Stále však můžete trasy nebo seznamy zveřejňovat.';

  @override
  String get settings_privacy_account_public =>
      'Váš profil může vidět kdokoli. Budete se zobrazovat ve výsledcích vyhledávání. Ostatní uživatelé vás mohou sledovat a sdílet s vámi trasy.';

  @override
  String get settings_privacy_lists_private =>
      'Vaše seznamy jsou ve výchozím nastavení soukromé. Nikdo kromě vás je neuvidí. Toto nastavení můžete u jednotlivých seznamů kdykoli změnit.';

  @override
  String get settings_privacy_lists_public =>
      'Vaše seznamy jsou ve výchozím nastavení veřejné. Uvidí je kdokoli. Toto nastavení můžete u jednotlivých seznamů kdykoli změnit.';

  @override
  String get settings_privacy_trails_private =>
      'Vaše trasy jsou ve výchozím nastavení soukromé. Nikdo kromě vás je neuvidí. Toto nastavení můžete u jednotlivých tras kdykoli změnit.';

  @override
  String get settings_privacy_trails_public =>
      'Vaše trasy jsou ve výchozím nastavení veřejné. Uvidí je kdokoli. Toto nastavení můžete u jednotlivých tras kdykoli změnit.';

  @override
  String get share => 'Sdílet';

  @override
  String get share_profile => 'Sdílet profil';

  @override
  String get shared => 'Sdíleno';

  @override
  String get show_on_map => 'Zobrazit na mapě';

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
  String get slogan => 'Vaše trasy. Vaše data. Váš server.';

  @override
  String get sort => 'Seřadit';

  @override
  String get speed => 'Rychlost';

  @override
  String get start => 'Start';

  @override
  String get subcategories => 'Podkategorie';

  @override
  String get summit_book => 'Vrcholová kniha';

  @override
  String get sync_failed => 'Upload failed · Tap to retry';

  @override
  String get sync_pending => 'Waiting to upload';

  @override
  String get sync_uploading => 'Uploading…';

  @override
  String get tags => 'Štítky';

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
      other: 'Tras',
      few: 'Trasy',
      one: 'Trasa',
    );
    return '$_temp0';
  }

  @override
  String get trail_saved_successfully => 'Trasa úspěšně uložena';

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
  String get units => 'Jednotky';

  @override
  String get users => 'Users';

  @override
  String get username => 'Uživatelské jméno';

  @override
  String get username_not_unique =>
      'Toto uživatelské jméno je již obsazeno. Zkuste prosím jiné.';

  @override
  String get visibilty_status => 'Stav viditelnosti';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Bodů trasy',
      few: 'Body trasy',
      one: 'Bod trasy',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get wrong_username_or_password =>
      'Nesprávné uživatelské jméno nebo heslo';

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
  String get open_in_new_tab => 'Otevřít na nové kartě';

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
  String get reverse_direction => 'Obrátit směr';

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
  String get edit_route => 'Upravit trasu';

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
