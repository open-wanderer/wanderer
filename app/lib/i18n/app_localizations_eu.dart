// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Basque (`eu`).
class AppLocalizationsEu extends AppLocalizations {
  AppLocalizationsEu([String locale = 'eu']) : super(locale);

  @override
  String get biking => 'Biking';

  @override
  String get canoeing => 'Canoeing';

  @override
  String get walking => 'Walking';

  @override
  String get about => 'Honi buruz';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Zure kontua ezabatzera zoaz. Zure ibilbide guztiak ere ezabatu egingo dira. Jarraitu nahi duzu?';

  @override
  String get account_privacy => 'Kontuaren pribatutasuna';

  @override
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'ekintza',
      one: 'ekintza',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => 'Gehitu bio';

  @override
  String get add_entry => 'Gehitu sarrera';

  @override
  String get add_to_list => 'Gehitu zerrendan';

  @override
  String get add_waypoint => 'Gehitu bidepuntua';

  @override
  String get added_trail_to => 'Ibilbidea hona gehitu da';

  @override
  String get added_trails_to => 'Ibilbideak hona gehitu dira';

  @override
  String get after => 'Ondoren';

  @override
  String get all => 'All';

  @override
  String get all_activities => 'Ekintza guztiak';

  @override
  String get allow_auto_geolocate =>
      'Begin drawing a new trail from your current location';

  @override
  String get alphabetical => 'Alfabetikoa';

  @override
  String get already_account => 'Baduzu kontua lehendik?';

  @override
  String get altitude => 'Altuera';

  @override
  String get amenity => 'Altimetria';

  @override
  String get api_documentation => 'API dokumentazioa';

  @override
  String get api_tokens => 'API Tokens';

  @override
  String get api_tokens_hint =>
      'API Tokens can be used to grant 3rd party applications access to your wanderer account.';

  @override
  String get apply_user_settings => 'Apply user settings';

  @override
  String get attraction => 'Erakarmena';

  @override
  String get author => 'Egilea';

  @override
  String get avatar => 'Iruditxoa';

  @override
  String get average_speed => 'Batazbesteko abiadura';

  @override
  String get avoid_bad_surfaces => 'Gainazal txarrak ekidin';

  @override
  String get back => 'Atzera';

  @override
  String get back_to_login => 'Itzuli saio hasierara';

  @override
  String get bakery => 'Okindegia';

  @override
  String get barrier => 'Oztopoa';

  @override
  String get basic_info => 'Oinarrizko informazioa';

  @override
  String get basque => 'Euskara';

  @override
  String get before => 'Aurretik';

  @override
  String get behavior => 'Behavior';

  @override
  String get bicycle_parking => 'Bizikleta-parkina';

  @override
  String get bicycle_rental => 'Bizikleta-alokairua';

  @override
  String get bicycle_shop => 'Bizikleta-denda';

  @override
  String get bike_type => 'Bizikleta mota';

  @override
  String get bus_stop => 'Autobus geltokia';

  @override
  String get by => '-';

  @override
  String get campsite => 'Kanpina';

  @override
  String get can => 'ahal duzu';

  @override
  String get cancel => 'Utzi';

  @override
  String get car => 'Kotxea';

  @override
  String get car_motorcycle => 'Kotxea/Motozikleta';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'txartel',
      one: 'txartel',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Kategoriak';

  @override
  String get category => 'Kategoria';

  @override
  String get change => 'Aldatu';

  @override
  String get change_email => 'Aldatu eposta';

  @override
  String get change_password => 'Aldatu pasahitza';

  @override
  String get changelog => 'Aldaketen erregistroa';

  @override
  String get chinese => 'Txinera (sinplifikatua)';

  @override
  String get clear_all => 'Garbitu dena';

  @override
  String get climbing => 'Eskalada';

  @override
  String get close => 'Itxi';

  @override
  String get collapse_trail_list => 'Collapse trail list';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'iruzkin',
      one: 'iruzkin',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Osatuta';

  @override
  String get completed_a_trail => 'ibilbide bat egin du';

  @override
  String get completed_tours => 'Egindako ibilbideak';

  @override
  String get completion_status => 'Betetze-egoera';

  @override
  String get confirm => 'Berretsi';

  @override
  String get confirm_deletion => 'Baieztatu ezabapena';

  @override
  String get confirm_publish => 'Baieztatu argitaratzea';

  @override
  String get confirm_share => 'Baieztatu partekatzea';

  @override
  String get connect => 'Konektatu';

  @override
  String get contribute => 'Ekarpena egin';

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
  String get copy_link => 'Kopiatu Lotura';

  @override
  String get create_new_list => 'Sortu zerrenda berria';

  @override
  String get create_waypoint => 'Sortu bidepuntua';

  @override
  String get creation_date => 'Sorrera-data';

  @override
  String get crop => 'Moztu';

  @override
  String get cross => 'Krossa';

  @override
  String get current_password => 'Oraingo pasahitza';

  @override
  String get cycling => 'Txirrindularitza';

  @override
  String get cycling_speed => 'Abiadura Bizikletan';

  @override
  String get czech => 'Czech';

  @override
  String get danger_zone => 'Arrisku-eremua';

  @override
  String get date => 'Data';

  @override
  String get default_category => 'Defektuzko kategoria';

  @override
  String get default_location => 'Defektuzko kokalekua';

  @override
  String get degrees => 'Graduak';

  @override
  String get delete => 'Ezabatu';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Ezabako kontua';

  @override
  String get delete_list_confirm =>
      'Benetan zerrenda hau ezabatu nahi duzu? Bertako ibilbideak ez dira ezabatuko.';

  @override
  String get delete_summit_log_confirm =>
      'Gailurreko liburua ezabatu nahi duzu? Akzio hau ezin da atzera bota.';

  @override
  String get delete_trail_confirm =>
      'Ibilbide hau ezabatu nahi duzu? Akzio hau ezin da atzera bota.';

  @override
  String get describe_your_trail => 'Deskribatu zure ibilbidea';

  @override
  String get description => 'Deskribapena';

  @override
  String get difficult => 'Zaila';

  @override
  String get difficulty => 'Zailtasuna';

  @override
  String get directions => 'Norabideak';

  @override
  String get display => 'Erakutsi';

  @override
  String get display_as => 'Horrela erakutsi';

  @override
  String get distance => 'Distantzia';

  @override
  String get documentation => 'Dokumentazioa';

  @override
  String get download => 'Deskargatu';

  @override
  String get draw_a_route => 'Marraztu ibilbide bat';

  @override
  String get driving => 'Gidatzen';

  @override
  String get duplicate => 'Bizkoiztu';

  @override
  String get duration => 'Iraupena';

  @override
  String get dutch => 'Nederlandera';

  @override
  String get easy => 'Erraza';

  @override
  String get edit => 'Editatu';

  @override
  String get edit_entry => 'Editatu sarrera';

  @override
  String get edit_list => 'Editatu zerrenda';

  @override
  String get edit_route => 'Editatu ibilbidea';

  @override
  String get edit_waypoint => 'Editatu bidepuntua';

  @override
  String get edited => 'editatuta';

  @override
  String get elevation_gain => 'Irabazitako altitudea';

  @override
  String get elevation_loss => 'Galdutako altitudea';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => 'Eposta';

  @override
  String get email_not_unique => 'This email address is already in use.';

  @override
  String get email_updated => 'Eposta eguneratuta';

  @override
  String get email_verified => 'Eposta egiaztatuta';

  @override
  String empty_activities(Object username) {
    return '$username erabiltzaileak ez du ekintzarik oraindik';
  }

  @override
  String empty_bio(Object username) {
    return '$username erabiltzaileak ez du biografiarik gehitu';
  }

  @override
  String get empty_feed => 'Zure jarioa hutsik dago';

  @override
  String get empty_feed_explanation =>
      'Zure ekintzak edo jarraitzen dituzun pertsonenak hemen agertuko dira';

  @override
  String empty_lists(Object username) {
    return '$username erabiltzailek ez du zerrenda publikorik';
  }

  @override
  String get enable_auto_routing => 'Aktibatu bideratze automatikoa';

  @override
  String get english => 'Ingelesa';

  @override
  String get entry => 'Sarrera';

  @override
  String get error_copying_trail => 'Error copying trail';

  @override
  String get error_creating_user => 'Errorea erabiltzailea sortzen';

  @override
  String get error_deleting_token => 'Error deleting token';

  @override
  String get error_disabling_strava_integration =>
      'Errorea stravarekin integrazioa desaktibatzean';

  @override
  String get error_during_login => 'Errorea sartzean';

  @override
  String get error_during_password_reset =>
      'Ezin izan da pasahitza berrezartzeko mezua bidali';

  @override
  String get error_exporting_trail => 'Errorea ibilbidea esportatzean';

  @override
  String get error_generating_token => 'Error generating token';

  @override
  String get error_liking_trail => 'Errorea ibilbidea atsegitean';

  @override
  String get error_logging_in_to_hammerhead => 'Error logging in to Hammerhead';

  @override
  String get error_logging_in_to_komoot => 'Errorea komoot-en login egitean';

  @override
  String get error_posting_comment => 'Errorea iruzkina egitean';

  @override
  String get error_printing_map => 'Errorea mapa inprimatzean';

  @override
  String get error_reading_file => 'Errorea fitxategia irakurtzean';

  @override
  String get error_saving_list => 'Errorea zerrenda gordetzean';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Errorea ibilbidea gordetzean';

  @override
  String error_setting_up_integration(Object provider) {
    return 'Errorea $provider integrazioa egitean';
  }

  @override
  String get error_updating_hammerhead_integration =>
      'Error updating Hammerhead integration';

  @override
  String get error_updating_komoot_integration =>
      'Error updating komoot integration';

  @override
  String get error_updating_password => 'Errorea pasahitza eguneratzean';

  @override
  String get error_updating_strava_integration =>
      'Errorea komoot integrazioa eguneratzean';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Error uploading trail to Hammerhead';

  @override
  String get est_duration => 'Ustezko iraupena';

  @override
  String get everyone_with_the_link => 'Esteka duen edonor';

  @override
  String get expand_trail_list => 'Expand trail list';

  @override
  String get expiration => 'Expiration';

  @override
  String get expires => 'Expires';

  @override
  String get explore => 'Arakatu';

  @override
  String get explore_some_trails => 'Arakatu ibilbide batzuk';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get stop_recording => 'Stop recording';

  @override
  String get stop_recording_confirm => 'Stop recording?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get export => 'Esportatu';

  @override
  String get export_all_trails => 'Esportatu ibilbide guztiak';

  @override
  String get favourite_sport => 'Kirol gogokoena';

  @override
  String get features => 'Ezaugarriak';

  @override
  String get ferry => 'Ferria';

  @override
  String get file_format => 'Fitxategiaren formatua';

  @override
  String file_too_big(Object file, Object size) {
    return '$file fitxategia handiegia da (max. $size)';
  }

  @override
  String get filter_categories => 'Iragazi kategoriak';

  @override
  String get filter_difficulty => 'Iragazi zailtasuna';

  @override
  String get filter_tags => 'Iragazi etiketak';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => 'Amaiera';

  @override
  String get finish_disabled_hint =>
      'Add at least 2 anchors to finish your route.';

  @override
  String get fixed_speed => 'Abiadura finkoa';

  @override
  String get focus_map_on => 'Kokatu mapa hemen';

  @override
  String get follow => 'Jarraitu';

  @override
  String get follow_request_pending => 'Eskaera zain';

  @override
  String get followers => 'Jarraitzaileak';

  @override
  String get following => 'Jarraitutakoak';

  @override
  String get food => 'Janaria';

  @override
  String get food_drinks => 'Janari eta edaria';

  @override
  String get forgot_your_password => 'Zure pasahitza ahaztu duzu?';

  @override
  String get french => 'Frantsesa';

  @override
  String get from_file => 'Fitxategitik';

  @override
  String get from_photos => 'Argazkietatik';

  @override
  String get from_url => 'URL batetik';

  @override
  String get garage => 'Garajea';

  @override
  String get gas_station => 'Gasolindegia';

  @override
  String get generate_new_token => 'Generate new token';

  @override
  String get german => 'Alemaniera';

  @override
  String get get_position_from_exif => 'Lortu koordenatuak EXIF datuetatik';

  @override
  String get get_started => 'Hasi';

  @override
  String get grid => 'Sareta';

  @override
  String get grocery_store => 'Janari-denda';

  @override
  String get hammerhead_integration_after_date_hint =>
      'If your hammerhead account is already synced with other trail databases, such as komoot or Strava, start syncing your Hammerhead data may result in duplicates. To avoid this, you can set an start date below, meaning only activities recorded after this date will be synced.';

  @override
  String get heading => 'Goiburukoa';

  @override
  String get height => 'Altuera';

  @override
  String get help => 'Laguntza';

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
  String get hiking => 'Mendi-ibilaldia';

  @override
  String get home => 'Hasiera';

  @override
  String get hotel => 'Hotela';

  @override
  String get hungarian => 'Hungariera';

  @override
  String get hut => 'Aterpea';

  @override
  String get hybrid => 'Hibridoa';

  @override
  String get icon => 'Ikonoa';

  @override
  String get ignore_trails_before_date => 'Ignore trails before this date';

  @override
  String get imperial => 'Inperiala';

  @override
  String get import => 'Inportatu';

  @override
  String get import_hint =>
      'Aukeratu edo arrastatu hona GPX, FIT, KML edo TCX fitxategiak...';

  @override
  String get include_description => 'Gehitu deskribapena';

  @override
  String get include_waypoints => 'Gehitu bidepuntuak';

  @override
  String get integration_description_hammerhead =>
      'Syncs your Hammerhead tours with wanderer in regular intervals.';

  @override
  String get integration_description_komoot =>
      'Zure komooteko ibilbideak wandererekin sinkronizatzen ditu aldian behin.';

  @override
  String get integration_description_strava =>
      'Zure stravako ibilbideak wandererekin sinkronizatzen ditu aldian behin.';

  @override
  String get integration_disabled => 'integrazioa desaktibatuta';

  @override
  String get integration_enabled => 'integrazioa aktibatuta';

  @override
  String get integration_privacy_hint_original =>
      'Imported trails will maintain the same visibility they have on the external platform. For example, if the original trail was public, it will be public in wanderer, even if trails are private by default according to your privacy settings.';

  @override
  String get integration_privacy_hint_user =>
      'The original trail\'s visibility is discarded. Instead, the local privacy settings for trails are applied to all imported trails.';

  @override
  String get integrations => 'Integrazioak';

  @override
  String get invalid_date => 'Data ez da zuzena';

  @override
  String get invalid_username => 'Erabiltzailea ez da zuzena';

  @override
  String get italian => 'Italiera';

  @override
  String get joined => 'Sartu da';

  @override
  String get keep_original => 'Keep original';

  @override
  String get keep_private => 'Keep private';

  @override
  String get language => 'Hizkuntza';

  @override
  String get last_used => 'Last used';

  @override
  String get latitude => 'Latitudea';

  @override
  String layer(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'geruza',
      one: 'geruza',
    );
    return '$_temp0';
  }

  @override
  String get license => 'Lizentzia';

  @override
  String get like_status => 'Atsegin egoera';

  @override
  String get liked => 'Atseginda';

  @override
  String get likes => 'Atsegiteak';

  @override
  String get limited => 'Mugatuta';

  @override
  String get link_copied => 'Esteka kopiatu da!';

  @override
  String get linked_lists => 'Linked lists';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'zerrenda',
      one: 'zerrenda',
    );
    return '$_temp0';
  }

  @override
  String get list_not_shared => 'Inorekin partekatu gabe';

  @override
  String get list_public_warning =>
      'Zerreda honetako ibilbide guztiak publiko bihurtuko dira.';

  @override
  String get list_saved_successfully => 'Zerrenda ondo gorde da';

  @override
  String get list_share_warning =>
      'Zerrenda bat partekatzean, bertako ibilbide guztiak partekatuko dira.';

  @override
  String get list_share_warning_update =>
      'Gehitzen diren ibilbideak, zerrendara sarbidea duten guztiekin partekatuko dira.';

  @override
  String get location => 'Kokalekua';

  @override
  String get locations => 'Locations';

  @override
  String get center_on_my_location => 'Center on my location';

  @override
  String get location_tracking_notification_title => 'Wanderer';

  @override
  String get location_tracking_notification_text => 'Recording your trail';

  @override
  String get login => 'Hasi saioa';

  @override
  String get login_details => 'Sartzeko xehetasunak';

  @override
  String get logout => 'Irten';

  @override
  String get longitude => 'Longitudea';

  @override
  String get loop => 'Begizta';

  @override
  String get make_one => 'Egin bat!';

  @override
  String get make_thumbnail => 'Egin iruditxoa';

  @override
  String get map => 'Mapa';

  @override
  String get map_style => 'Maparen estiloa';

  @override
  String get max_hiking_difficulty =>
      'Mendi ibilaldiaren gehienezko zailtasuna';

  @override
  String get metric => 'Metrikoa';

  @override
  String get moderate => 'Erdi-bidekoa';

  @override
  String get more => 'Gehiago';

  @override
  String get more_route_settings => 'Ibilbidearen ezarpen gehiago';

  @override
  String get mountain => 'Mendia';

  @override
  String get mountain_pass => 'Mendiko pasabidea';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return 'Gehienez $n karaktere izan behar ditu';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return 'Gehienez $n karaktere izan behar ditu';
  }

  @override
  String get my_account => 'Nire kontua';

  @override
  String get my_trails => 'Nire ibilbideak';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return 'Orain dela $n egun';
  }

  @override
  String n_hours_ago(Object n) {
    return 'Orain dela $n ordu';
  }

  @override
  String n_minutes_ago(Object n) {
    return 'Orain dela $n minutu';
  }

  @override
  String n_months_ago(Object n) {
    return 'Orain dela $n hilabete';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'orain dela $n segundo',
      zero: 'oraintxe',
      one: '',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return 'Orain dela $n urte';
  }

  @override
  String get name => 'Izena';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => 'Gertu';

  @override
  String get never => 'Never';

  @override
  String get new_list => 'Zerrenda berria';

  @override
  String get new_password => 'Pasahitz berria';

  @override
  String get new_password_error => 'Errorea pasahitza berria ezartzean';

  @override
  String get new_password_success => 'Pasahitz berria ezarri da';

  @override
  String get new_password_text => 'Pasahitz berria sortu';

  @override
  String get new_token_generated => 'New API Token generated';

  @override
  String get new_trail => 'Ibilaldi berria';

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
  String get no_account => 'Ez duzu konturik?';

  @override
  String get no_api_tokens => 'You have no API Tokens';

  @override
  String get no_comments_so_far => 'Ez dago iruzkinik';

  @override
  String get no_data => 'Ez dago daturik';

  @override
  String get no_description_for_now => 'Ez dago deskribapenik';

  @override
  String get no_gps_data_in_image => 'Ez dago GPS daturik irudian';

  @override
  String get no_grid => 'Ez dago saretarik';

  @override
  String get no_notifications => 'Ez dago jakinarazpenik';

  @override
  String get no_photos_here => 'Ez dago argazki edo bideorik';

  @override
  String get no_preference => 'Ez dago lehentasunik';

  @override
  String get no_results => 'Ez dago emaitzarik';

  @override
  String get no_routes_added => 'Ez da ibilaldirik gehitu';

  @override
  String get no_waypoints_yet => 'Ez dago bidepunturik';

  @override
  String get norwegian => 'Norwegian';

  @override
  String get not_a_valid_email_address => 'Ez da eposta helbide zuzena';

  @override
  String get not_a_valid_url => 'Ez da URL zuzena';

  @override
  String get not_completed => 'Ez dago osatuta';

  @override
  String notification_comment_mention(Object user) {
    return '$user erabiltzaileak iruzkin baten aipatu zaitu';
  }

  @override
  String notification_list_create(Object user) {
    return '$user erabiltzaileak zerrenda berria sortu du';
  }

  @override
  String notification_list_share(Object user) {
    return '$user erabiltzaileak zerrenda partekatu du zurekin';
  }

  @override
  String get notification_new_follower => 'Jarraitzaile berri bat duzu';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user erabiltzaileak igoeren liburua sortu du zure \"$trail\" ibilbidean';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user erabiltzaileak gailurreko liburu baten aipatu zaitu';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user erabiltzaileak iruzkina utzi du zure \"$trail\" ibilbidean';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user erabiltzaileak ibilbide berria sortu du';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user erabiltzaileak zure \"$trail\" ibilbidea atsegin du';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user erabiltzaileak ibilbide baten aipatu zaitu';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user erabiltzaileak ibilbide bat partekatu du zurekin';
  }

  @override
  String get notifications => 'Jakinarazpenak';

  @override
  String object_share_error(Object object) {
    return '$object publikoa izan behar da instantzien artean partekatzeko.';
  }

  @override
  String get off => 'Itzalita';

  @override
  String get only_me => 'Ni bakarrik';

  @override
  String get open_in_new_tab => 'Ireki fitxa berrian';

  @override
  String get or => 'edo';

  @override
  String get orientation => 'Orientazioa';

  @override
  String get paper_size => 'Paperaren tamaina';

  @override
  String get paragraph => 'Paragrafoa';

  @override
  String get parking => 'Aparkalekua';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Pasahitza';

  @override
  String get password_confirm => 'Berretsi pasahitza';

  @override
  String get password_reset_sent =>
      'Pasahitza berrezartzeko eposta mezua bidali dizugu';

  @override
  String get password_reset_text =>
      'Zure epostara berrezartzeko esteka bat bidaliko dizugu.';

  @override
  String get password_updated => 'Pasahitza eguneratu da';

  @override
  String get passwords_must_match => 'Pasahitzek bat etorri behar dute';

  @override
  String get photos => 'Argazkiak eta bideoak';

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
  String get pick_a_trail => 'Aukeratu ibilaldi bat';

  @override
  String get planned_a_trail => 'ibilaldi bat planifikatu du';

  @override
  String get planned_tours => 'Planifikatutako ibilaldiak';

  @override
  String get pois => 'Interes-puntuak';

  @override
  String get polish => 'Poloniera';

  @override
  String get portuguese => 'Portugalera';

  @override
  String get print => 'Inprimatu';

  @override
  String get privacy => 'Pribatutasuna';

  @override
  String get private => 'Pribatua';

  @override
  String get profile => 'Profila';

  @override
  String get public => 'Publikoa';

  @override
  String get public_access => 'Sarbide publikoa';

  @override
  String get public_share_everyone =>
      'Esteka duen edonork ikusi dezake ibilbide hau';

  @override
  String get public_share_limited =>
      'Sarbidea duten pertsonek bakarrik ireki dezakete esteka hau';

  @override
  String get public_transport => 'Garraio publikoa';

  @override
  String get radius => 'Erradioa';

  @override
  String get railway_station => 'Tren geltokia';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get read_more => 'Irakurri gehiago';

  @override
  String get ready_to_join => 'Sartzeko prest';

  @override
  String get recalculate_elevation_data => 'Birkalkulatu altuera-datuak';

  @override
  String get recalculating_elevation_data_hint =>
      'Birkalkulatzean uneko datuak ezabatu egingo dira eta Valhallatik ekarritako datuekin ordezkatuko dira.';

  @override
  String get register => 'Izena eman';

  @override
  String get remote_users_cannot_edit =>
      'Urruneko erabiltzaileek ezin dute editatu';

  @override
  String get removed_trail_from => 'Ibilbidea ezabatu egin da';

  @override
  String get removed_trails_from => 'Ibilbideak ezabatu egin dira';

  @override
  String get required => 'Beharrezkoa';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Berrezarri';

  @override
  String get reset_password => 'Berrezarri pasahitza';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => 'Resume recording?';

  @override
  String get reverse_direction => 'Alderantzikatu norabidea';

  @override
  String get road => 'Errepidea';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'ibilbide',
      one: 'ibilbide',
    );
    return '$_temp0';
  }

  @override
  String get route_point => 'Ibilbideko puntua';

  @override
  String get russian => 'Errusiera';

  @override
  String get save => 'Gorde';

  @override
  String get save_list => 'Gorde zerrenda';

  @override
  String get save_track => 'Save track';

  @override
  String get save_trail => 'Gorde ibilaldia';

  @override
  String get save_your_trail_first => 'Gorde zure ibilaldia lehenengo';

  @override
  String get search => 'Search';

  @override
  String get search_cities => 'Bilatu herriak';

  @override
  String get search_for_trails_places => 'Bilatu ibilbideak, zerrendak, tokiak';

  @override
  String get search_list => 'Search list';

  @override
  String get search_places => 'Bilatu tokiak';

  @override
  String get search_trails => 'Bilatu ibilaldiak';

  @override
  String get select_date => 'Select date';

  @override
  String get select_list => 'Aukeratu zerrenda';

  @override
  String get selected => 'aukeratuta';

  @override
  String get send_to => 'Send to...';

  @override
  String get set_private => 'Set private';

  @override
  String get set_public => 'Set public';

  @override
  String get settings => 'Ezarpenak';

  @override
  String get settings_notification_comment_mention =>
      'Norbaitek iruzkin baten aipatu zaitu';

  @override
  String get settings_notification_list_share =>
      'Norbaitek zerrenda bat partekatu du zurekin';

  @override
  String get settings_notification_new_follower =>
      'Jarraitzaile berri bat duzu';

  @override
  String get settings_notification_summit_log_create =>
      'Norbaitek igoeren liburua sortu du zure ibilbidean';

  @override
  String get settings_notification_summit_log_mention =>
      'Norbaitek igoeren liburu baten aipatu zaitu';

  @override
  String get settings_notification_trail_comment =>
      'Norbaitek iruzkina utzi du zure ibilbide baten';

  @override
  String get settings_notification_trail_like => 'Somone liked your trail';

  @override
  String get settings_notification_trail_mention =>
      'Norbaitek ibilbide baten aipatu zaitu';

  @override
  String get settings_notification_trail_share =>
      'Norbaitek ibilbide bat partekatu du zurekin';

  @override
  String get settings_privacy_account_private =>
      'Zuk bakarrik ikusi dezakezu zure profila. Ez da bilaketa-emaitzetan agertuko. Beste erabiltzaileek ezin zaitutzte jarraitu edo ibilbiderik zurekin partekatu. Hala ere. ibilbideak edo zerrendak argitaratu ditzakezu.';

  @override
  String get settings_privacy_account_public =>
      'Edonork ikusi dezake zure profila. Bilaketa-emaitzetan agertuko zara. Beste erabiltzaileek jarraitu zaitzakete eta ibilbideak zurekin partekatu ditzakete.';

  @override
  String get settings_privacy_lists_private =>
      'Zure zerrendak pribatuak dira defektuz. Zuk bakarrik ikusi ditzakezu. Ezarpen hau zerrenda bakoitzean banan-banan aldatu dezakezu.';

  @override
  String get settings_privacy_lists_public =>
      'Zure zerrendak publikoak dira defektuz. Edonork ikusi ditzake. Ezarpen hau zerrenda bakoitzean banan-banana aldatu dezakezu.';

  @override
  String get settings_privacy_trails_private =>
      'Zure ibilbideak pribatuak dira defektuz. Zuk bakarrik ikusi ditzakezu. Ezarpen hau ibilbide bakoitzean banan-banan aldatu dezakezu.';

  @override
  String get settings_privacy_trails_public =>
      'Zure ibilbideak publikoak dira defektuz. Edonork ikusi ditzake. Ezarpen hau ibilbide bakoitzean banan-banana aldatu dezakezu.';

  @override
  String get settings_saved => 'Ezarpenak gorde dira';

  @override
  String get share => 'Partekatu';

  @override
  String get share_profile => 'Partekatu profila';

  @override
  String get share_this_list => 'Partekatu zerrenda hau';

  @override
  String get share_this_trail => 'Partekatu ibilaldi hau';

  @override
  String get shared => 'Partekatuta';

  @override
  String get shared_by => 'Honek partekatuta';

  @override
  String get shared_with => 'Honekin partekatuta';

  @override
  String get shelter => 'Babeslekua';

  @override
  String get shortest => 'laburrena';

  @override
  String get show_in_overview => 'Erakutsi laburpenean';

  @override
  String get show_less => 'Erakutsi gutxiago';

  @override
  String get show_on_map => 'Erakutsi mapan';

  @override
  String get shower => 'Dutxa';

  @override
  String get skiing => 'Eskia';

  @override
  String get slogan => 'Gorde zure abenturak!';

  @override
  String get slope => 'Malda';

  @override
  String get someone => 'Norbait';

  @override
  String get sort => 'Ordenatu';

  @override
  String get spanish => 'Espainiera';

  @override
  String get speed => 'Abiadura';

  @override
  String get start => 'Hasi';

  @override
  String get statistics => 'Estatistikak';

  @override
  String get stop_drawing => 'Utzi marrazteari';

  @override
  String get stop_editing => 'Utzi editatzeari';

  @override
  String get strava_integration_after_date_hint =>
      'Zure kontuak ekintza esko baditu Stravaren APIaren mugekin topo egin dezakezu eta agian ezingo dituzu zure ekintza guztiak aldi berean inportatu. Horretarako data jakin batetik aurrerako ekintzak sinkronizatzeko aukera duzu.';

  @override
  String get subcategories => 'Azpikategoriak';

  @override
  String get subway_stop => 'Metro sarbidea';

  @override
  String get summit => 'Gailurra';

  @override
  String get summit_book => 'Igoeren liburua';

  @override
  String get table => 'Taula';

  @override
  String get tags => 'Etiketak';

  @override
  String get text => 'Testua';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get tilesets => 'Maparen hondo pertsonalizatua';

  @override
  String get time => 'Time';

  @override
  String get time_in_motion => 'Time in Motion';

  @override
  String get toilets => 'Komunak';

  @override
  String get top_speed => 'Gehienezko abiadura';

  @override
  String get tourism => 'Turismoa';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'ibilbide',
      one: 'ibilbide',
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
  String get trail_not_shared => 'Inorekin partekatu gabe';

  @override
  String get trail_saved_successfully => 'Ibilbidea ondo gorde da';

  @override
  String get some_waypoints_failed_to_save =>
      'Trail saved, but some waypoints failed to save';

  @override
  String get trails_for_you => 'Zuretzako ibilbideak';

  @override
  String get tram_stop => 'Tranbia geltokia';

  @override
  String get unchanged => 'ez da aldatu';

  @override
  String get units => 'Unitateak';

  @override
  String get unlink => 'Esteka kendu';

  @override
  String get upload_file => 'Kargatu fitxategia';

  @override
  String get upload_gpx => 'Kargatu GPX';

  @override
  String get upload_new_file => 'Kargatu fitxategi berria';

  @override
  String get uploaded => 'kargatuta';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Successfully uploaded trail to Hammerhead';

  @override
  String get use_hills => 'Erabili aldapak';

  @override
  String get use_roads => 'Erabili errepideak';

  @override
  String get users => 'Users';

  @override
  String get username => 'Erabiltzaile-izena';

  @override
  String get username_not_unique =>
      'This username is already taken. Please try another.';

  @override
  String get view => 'Ikusi';

  @override
  String get viewpoint => 'Ikuspegia';

  @override
  String get visibilty => 'Visibility';

  @override
  String get visibilty_status => 'Ikuspen-egoera';

  @override
  String get walking_speed => 'Ibiltze abiadura';

  @override
  String get water => 'Ura';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'bidepuntu',
      one: 'bidepuntu',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get width => 'Zabalera';

  @override
  String get wrong_username_or_password =>
      'Erabiltzaile izena edo pasahitza okerrak dira';

  @override
  String get you_can => 'Hau egin dezakezu';

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
