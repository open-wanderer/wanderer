// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Czech (`cs`).
class AppLocalizationsCs extends AppLocalizations {
  AppLocalizationsCs([String locale = 'cs']) : super(locale);

  @override
  String get biking => 'Biking';

  @override
  String get canoeing => 'Canoeing';

  @override
  String get walking => 'Walking';

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
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Aktivit',
      few: 'Aktivity',
      one: 'Aktivita',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => 'Přidat informace o mně';

  @override
  String get add_entry => 'Přidat záznam';

  @override
  String get add_to_list => 'Přidat na seznam';

  @override
  String get add_waypoint => 'Přidat bod trasy';

  @override
  String get added_trail_to => 'Přidána trasa do';

  @override
  String get added_trails_to => 'Přidány trasy do';

  @override
  String get after => 'Po';

  @override
  String get all => 'All';

  @override
  String get all_activities => 'Všechny aktivity';

  @override
  String get allow_auto_geolocate =>
      'Begin drawing a new trail from your current location';

  @override
  String get alphabetical => 'Abecedně';

  @override
  String get already_account => 'Už máte svůj účet?';

  @override
  String get altitude => 'Nadmořská výška';

  @override
  String get amenity => 'Amenity';

  @override
  String get api_documentation => 'API dokumentace';

  @override
  String get api_tokens => 'API Tokens';

  @override
  String get api_tokens_hint =>
      'API Tokens can be used to grant 3rd party applications access to your wanderer account.';

  @override
  String get apply_user_settings => 'Apply user settings';

  @override
  String get attraction => 'Zajímavost';

  @override
  String get author => 'Autor';

  @override
  String get avatar => 'Avatar';

  @override
  String get average_speed => 'Průměrná rychlost';

  @override
  String get avoid_bad_surfaces => 'Vyhnout se špatnému povrchu';

  @override
  String get back => 'Zpět';

  @override
  String get back_to_login => 'Zpět na přihlášení';

  @override
  String get bakery => 'Pekařství';

  @override
  String get barrier => 'Překážka';

  @override
  String get basic_info => 'Základní informace';

  @override
  String get basque => 'Baskičtina';

  @override
  String get before => 'Před';

  @override
  String get behavior => 'Behavior';

  @override
  String get bicycle_parking => 'Parkoviště pro kola';

  @override
  String get bicycle_rental => 'Půjčovna jízdních kol';

  @override
  String get bicycle_shop => 'Prodejna jízdních kol';

  @override
  String get bike_type => 'Typ jízdního kola';

  @override
  String get bus_stop => 'Autobusová zastávka';

  @override
  String get by => 'od';

  @override
  String get campsite => 'Kemp';

  @override
  String get can => 'můžete';

  @override
  String get cancel => 'Zrušit';

  @override
  String get car => 'Auto';

  @override
  String get car_motorcycle => 'Automobil/motocykl';

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
  String get change => 'Změnit';

  @override
  String get change_email => 'Změnit e-mail';

  @override
  String get change_password => 'Změnit heslo';

  @override
  String get changelog => 'Seznam změn';

  @override
  String get chinese => 'Čínština (zjednodušená)';

  @override
  String get clear_all => 'Vymazat vše';

  @override
  String get climbing => 'Horolezectví';

  @override
  String get close => 'Zavřít';

  @override
  String get collapse_trail_list => 'Collapse trail list';

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
  String get completed_a_trail => 'dokončil/a trasu';

  @override
  String get completed_tours => 'Dokončené výlety';

  @override
  String get completion_status => 'Stav dokončení';

  @override
  String get confirm => 'Potvrdit';

  @override
  String get confirm_deletion => 'Potvrdit odstranění';

  @override
  String get confirm_publish => 'Potvrdit zveřejnění';

  @override
  String get confirm_share => 'Potvrďte sdílení';

  @override
  String get connect => 'Připojit';

  @override
  String get contribute => 'Přispět';

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
  String get copy_link => 'Kopírovat odkaz';

  @override
  String get create_new_list => 'Vytvořit nový seznam';

  @override
  String get create_waypoint => 'Vytvořit bod trasy';

  @override
  String get creation_date => 'Datum vytvoření';

  @override
  String get crop => 'Oříznout';

  @override
  String get cross => 'Křížek';

  @override
  String get current_password => 'Současné heslo';

  @override
  String get cycling => 'Cyklistika';

  @override
  String get cycling_speed => 'Rychlost jízdy na kole';

  @override
  String get czech => 'Čeština';

  @override
  String get danger_zone => 'Nebezpečná zóna';

  @override
  String get date => 'Datum';

  @override
  String get default_category => 'Výchozí kategorie';

  @override
  String get default_location => 'Výchozí lokace';

  @override
  String get degrees => 'Stupně';

  @override
  String get delete => 'Vymazat';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Odstranit účet';

  @override
  String get delete_list_confirm =>
      'Opravdu chcete tento seznam smazat? Trasy budou v seznamu i nadále k dispozici.';

  @override
  String get delete_summit_log_confirm =>
      'Opravdu chcete smazat tento záznam o výstupu? Tuto akci nelze vrátit.';

  @override
  String get delete_trail_confirm =>
      'Doopravdy chcete tuto trasu smazat? Tuto akci nelze vrátit zpět.';

  @override
  String get describe_your_trail => 'Popište svou trasu';

  @override
  String get description => 'Popis';

  @override
  String get difficult => 'Náročné';

  @override
  String get difficulty => 'Obtížnost';

  @override
  String get directions => 'Navigovat';

  @override
  String get display => 'Zobrazit';

  @override
  String get display_as => 'Zobrazit jako';

  @override
  String get distance => 'Vzdálenost';

  @override
  String get documentation => 'Dokumentace';

  @override
  String get download => 'Stáhnout';

  @override
  String get draw_a_route => 'Nakreslete trasu';

  @override
  String get driving => 'Řízení';

  @override
  String get duplicate => 'Duplikovat';

  @override
  String get duration => 'Doba trvání';

  @override
  String get dutch => 'Holandština';

  @override
  String get easy => 'Jednoduchý';

  @override
  String get edit => 'Upravit';

  @override
  String get edit_entry => 'Upravit položku';

  @override
  String get edit_list => 'Upravit seznam';

  @override
  String get edit_route => 'Upravit trasu';

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
  String get email_not_unique => 'This email address is already in use.';

  @override
  String get email_updated => 'E-mail byl aktualizován';

  @override
  String get email_verified => 'E-mail byl ověřen';

  @override
  String empty_activities(Object username) {
    return '$username zatím nemá žádnou aktivitu';
  }

  @override
  String empty_bio(Object username) {
    return '$username zatím nepřidal/a informace o sobě';
  }

  @override
  String get empty_feed => 'Váš kanál novinek je prázdný';

  @override
  String get empty_feed_explanation =>
      'Zde se zobrazí vaše aktivity nebo aktivity lidí, které sledujete';

  @override
  String empty_lists(Object username) {
    return '$username nemá žádné veřejné seznamy';
  }

  @override
  String get enable_auto_routing => 'Zapnout automatické trasování';

  @override
  String get english => 'Angličtina';

  @override
  String get entry => 'Vstup';

  @override
  String get error_copying_trail => 'Error copying trail';

  @override
  String get error_creating_user => 'Chyba při vytváření uživatele';

  @override
  String get error_deleting_token => 'Error deleting token';

  @override
  String get error_disabling_strava_integration =>
      'Chyba při deaktivaci integrace strava';

  @override
  String get error_during_login => 'Během přihlášení došlo k chybě';

  @override
  String get error_during_password_reset =>
      'Nelze odeslat e-mail pro obnovu hesla';

  @override
  String get error_exporting_trail => 'Chyba při exportování trasy';

  @override
  String get error_generating_token => 'Error generating token';

  @override
  String get error_liking_trail => 'Chyba při lajkování trasy';

  @override
  String get error_logging_in_to_hammerhead => 'Error logging in to Hammerhead';

  @override
  String get error_logging_in_to_komoot => 'Chyba při přihlášení do Komootu';

  @override
  String get error_posting_comment => 'Chyba při vkládání komentáře';

  @override
  String get error_printing_map => 'Při tisku mapy došlo k chybě';

  @override
  String get error_reading_file => 'Soubor nelze načíst';

  @override
  String get error_saving_list => 'Chyba při ukládání seznamu';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Chyba při ukládání trasy';

  @override
  String error_setting_up_integration(Object provider) {
    return 'Chyba během nastavování $provider integrace';
  }

  @override
  String get error_updating_hammerhead_integration =>
      'Error updating Hammerhead integration';

  @override
  String get error_updating_komoot_integration =>
      'Error updating komoot integration';

  @override
  String get error_updating_password => 'Chyba během aktualizace hesla';

  @override
  String get error_updating_strava_integration =>
      'Chyba při aktualizaci Komoot integrace';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Error uploading trail to Hammerhead';

  @override
  String get est_duration => 'Odhadovaná doba trvání';

  @override
  String get everyone_with_the_link => 'Každý, kdo má odkaz';

  @override
  String get expand_trail_list => 'Expand trail list';

  @override
  String get expiration => 'Expiration';

  @override
  String get expires => 'Expires';

  @override
  String get explore => 'Objevování';

  @override
  String get explore_some_trails => 'Objevte trasy';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get export => 'Exportovat';

  @override
  String get export_all_trails => 'Exportovat všechny trasy';

  @override
  String get favourite_sport => 'Oblíbený sport';

  @override
  String get features => 'Vlastnosti';

  @override
  String get ferry => 'Přívoz';

  @override
  String get file_format => 'Formát souboru';

  @override
  String file_too_big(Object file, Object size) {
    return 'Soubor $file je příliš velký (max. $size)';
  }

  @override
  String get filter_categories => 'Filtrovat podle kategorií';

  @override
  String get filter_difficulty => 'Filtrovat podle obtížnosti';

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
  String get fixed_speed => 'Konstantní rychlost';

  @override
  String get focus_map_on => 'Zaměřit mapu na';

  @override
  String get follow => 'Sledovat';

  @override
  String get follow_request_pending => 'Žádost čeká na vyřízení';

  @override
  String get followers => 'Sledující';

  @override
  String get following => 'Sleduji';

  @override
  String get food => 'Jídlo';

  @override
  String get food_drinks => 'Jídlo a pití';

  @override
  String get forgot_your_password => 'Zapomněli jste heslo?';

  @override
  String get french => 'Francouzština';

  @override
  String get from_file => 'Ze souboru';

  @override
  String get from_photos => 'Z fotografií';

  @override
  String get from_url => 'Z URL adresy';

  @override
  String get garage => 'Garáž';

  @override
  String get gas_station => 'Benzínka';

  @override
  String get generate_new_token => 'Generate new token';

  @override
  String get german => 'Němčina';

  @override
  String get get_position_from_exif => 'Získat souřadnice z EXIF údajů';

  @override
  String get get_started => 'Začněte';

  @override
  String get grid => 'Mřížka';

  @override
  String get grocery_store => 'Potraviny';

  @override
  String get hammerhead_integration_after_date_hint =>
      'If your hammerhead account is already synced with other trail databases, such as komoot or Strava, start syncing your Hammerhead data may result in duplicates. To avoid this, you can set an start date below, meaning only activities recorded after this date will be synced.';

  @override
  String get heading => 'Nadpis';

  @override
  String get height => 'Výška';

  @override
  String get help => 'Nápověda';

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
  String get hiking => 'Turistika';

  @override
  String get home => 'Hlavní stránka';

  @override
  String get hotel => 'Hotel';

  @override
  String get hungarian => 'Maďarština';

  @override
  String get hut => 'Chatka';

  @override
  String get hybrid => 'Hybridní';

  @override
  String get icon => 'Ikonka';

  @override
  String get ignore_trails_before_date => 'Ignore trails before this date';

  @override
  String get imperial => 'Imperiální jednotky';

  @override
  String get import => 'Importovat';

  @override
  String get import_hint =>
      'Vyberte nebo sem přetáhněte soubory GPX, FIT, KML, či TCX...';

  @override
  String get include_description => 'Zahrnout popis';

  @override
  String get include_waypoints => 'Zahrnout body trasy';

  @override
  String get integration_description_hammerhead =>
      'Syncs your Hammerhead tours with wanderer in regular intervals.';

  @override
  String get integration_description_komoot =>
      'Synchronizuje vaše trasy z aplikace Komoot s Wandererem v pravidelných intervalech.';

  @override
  String get integration_description_strava =>
      'Synchronizuje vaše trasy a aktivity z aplikace Strava s Wandererem v pravidelných intervalech.';

  @override
  String get integration_disabled => 'integrace zakázána';

  @override
  String get integration_enabled => 'integrace povolena';

  @override
  String get integration_privacy_hint_original =>
      'Imported trails will maintain the same visibility they have on the external platform. For example, if the original trail was public, it will be public in wanderer, even if trails are private by default according to your privacy settings.';

  @override
  String get integration_privacy_hint_user =>
      'The original trail\'s visibility is discarded. Instead, the local privacy settings for trails are applied to all imported trails.';

  @override
  String get integrations => 'Integrace';

  @override
  String get invalid_date => 'Neplatné datum';

  @override
  String get invalid_username => 'Neplatné uživatelské jméno';

  @override
  String get italian => 'Italština';

  @override
  String get joined => 'Připojeno';

  @override
  String get keep_original => 'Keep original';

  @override
  String get keep_private => 'Keep private';

  @override
  String get language => 'Jazyk';

  @override
  String get last_used => 'Last used';

  @override
  String get latitude => 'Zeměpisná šířka';

  @override
  String layer(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Vrstev',
      few: 'Vrstvy',
      one: 'Vrstva',
    );
    return '$_temp0';
  }

  @override
  String get license => 'Licence';

  @override
  String get like_status => 'Status „Líbí se mi“';

  @override
  String get liked => 'Líbilo se';

  @override
  String get likes => 'Líbí se';

  @override
  String get limited => 'Omezené';

  @override
  String get link_copied => 'Odkaz zkopírován!';

  @override
  String get linked_lists => 'Linked lists';

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
  String get list_not_shared => 'Není s nikým sdíleno';

  @override
  String get list_public_warning =>
      'Všechny trasy v tomto seznamu budou veřejné.';

  @override
  String get list_saved_successfully => 'Seznam byl úspěšně uložen';

  @override
  String get list_share_warning =>
      'Sdílením seznamu se automaticky i sdílí všechny trasy, které jsou v něm obsaženy.';

  @override
  String get list_share_warning_update =>
      'Přidané trasy budou sdíleny se všemi, kteří mají přístup do tohoto seznamu.';

  @override
  String get location => 'Poloha';

  @override
  String get locations => 'Locations';

  @override
  String get location_tracking_notification_title => 'Wanderer';

  @override
  String get location_tracking_notification_text => 'Recording your trail';

  @override
  String get login => 'Přihlásit se';

  @override
  String get login_details => 'Přihlašovací údaje';

  @override
  String get logout => 'Odhlásit se';

  @override
  String get longitude => 'Zeměpisná délka';

  @override
  String get loop => 'Opakování';

  @override
  String get make_one => 'Vytvořte si vlastní!';

  @override
  String get make_thumbnail => 'Vytvořit náhled';

  @override
  String get map => 'Mapa';

  @override
  String get map_style => 'Styl mapy';

  @override
  String get max_hiking_difficulty => 'Max. obtížnost túry';

  @override
  String get metric => 'Metrický';

  @override
  String get moderate => 'Středně náročné';

  @override
  String get more => 'Více';

  @override
  String get more_route_settings => 'Další nastavení trasy';

  @override
  String get mountain => 'Hora';

  @override
  String get mountain_pass => 'Horský průsmyk';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return 'Musí být alespoň $n znaků dlouhé';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return 'Musí být maximálně $n znaků dlouhé';
  }

  @override
  String get my_account => 'Můj účet';

  @override
  String get my_trails => 'Moje trasy';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return 'před $n dny';
  }

  @override
  String n_hours_ago(Object n) {
    return 'před $n hodinami';
  }

  @override
  String n_minutes_ago(Object n) {
    return 'před $n minutami';
  }

  @override
  String n_months_ago(Object n) {
    return 'před $n měsíci';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'před $n sekundami',
      zero: 'právě teď',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return 'před $n lety';
  }

  @override
  String get name => 'Název';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => 'Poblíž';

  @override
  String get never => 'Never';

  @override
  String get new_list => 'Nový seznam';

  @override
  String get new_password => 'Nové heslo';

  @override
  String get new_password_error => 'Chyba při nastavení nového hesla';

  @override
  String get new_password_success => 'Nové heslo bylo nastaveno';

  @override
  String get new_password_text => 'Nastavit nové heslo';

  @override
  String get new_token_generated => 'New API Token generated';

  @override
  String get new_trail => 'Nová trasa';

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
  String get no_account => 'Ještě nemáte účet?';

  @override
  String get no_api_tokens => 'You have no API Tokens';

  @override
  String get no_comments_so_far => 'Zatím žádné komentáře';

  @override
  String get no_data => 'Žádná data';

  @override
  String get no_description_for_now => 'Zatím bez popisu';

  @override
  String get no_gps_data_in_image => 'Obrázek neobsahuje GPS data';

  @override
  String get no_grid => 'Bez mřížky';

  @override
  String get no_notifications => 'Žádná upozornění';

  @override
  String get no_photos_here => 'Zde nejsou žádné fotky ani videa';

  @override
  String get no_preference => 'Bez preference';

  @override
  String get no_results => 'Nebyly nalezeny žádné výsledky';

  @override
  String get no_routes_added => 'Nebyly přidány žádné trasy';

  @override
  String get no_waypoints_yet => 'Zatím žádné body trasy';

  @override
  String get norwegian => 'Norwegian';

  @override
  String get not_a_valid_email_address => 'Neplatná e-mailová adresa';

  @override
  String get not_a_valid_url => 'Neplatná URL adresa';

  @override
  String get not_completed => 'Nedokončeno';

  @override
  String notification_comment_mention(Object user) {
    return '$user vás zmínil/a v komentáři';
  }

  @override
  String notification_list_create(Object user) {
    return '$user vytvořil/a nový seznam';
  }

  @override
  String notification_list_share(Object user) {
    return '$user s vámi sdílel/a seznam';
  }

  @override
  String get notification_new_follower => 'Máte nového sledujícího';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user přidal/a záznam o výstupu na vaší trase \"$trail\"';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user vás zmínil/a v záznamu o výstupu';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user zanechal/a komentář u vaší trasy \"$trail\"';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user vytvořil/a novou trasu';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user se líbila vaše trasa \"$trail\"';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user vás zmínil/a u trasy';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user s vámi sdílel/a trasu';
  }

  @override
  String get notifications => 'Oznámení';

  @override
  String object_share_error(Object object) {
    return 'Objekt $object musí býti veřejný, aby se dal sdílet mezi instancemi.';
  }

  @override
  String get off => 'Vypnuto';

  @override
  String get only_me => 'Jen já';

  @override
  String get open_in_new_tab => 'Otevřít na nové kartě';

  @override
  String get or => 'nebo';

  @override
  String get orientation => 'Orientace';

  @override
  String get paper_size => 'Formát papíru';

  @override
  String get paragraph => 'Odstavec';

  @override
  String get parking => 'Parkoviště';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Heslo';

  @override
  String get password_confirm => 'Potvrďte heslo';

  @override
  String get password_reset_sent =>
      'E-mail pro obnovení Vašeho hesla byl odeslán';

  @override
  String get password_reset_text =>
      'Na vaši e-mailovou adresu zašleme odkaz pro obnovení.';

  @override
  String get password_updated => 'Heslo bylo úspěšně změněno';

  @override
  String get passwords_must_match => 'Hesla se musí shodovat';

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
  String get pick_a_trail => 'Vyberte trasu';

  @override
  String get planned_a_trail => 'naplánoval/a trasu';

  @override
  String get planned_tours => 'Plánované výlety';

  @override
  String get pois => 'Body zájmu';

  @override
  String get polish => 'Polština';

  @override
  String get portuguese => 'Portugalština';

  @override
  String get print => 'Vytisknout';

  @override
  String get privacy => 'Soukromí';

  @override
  String get private => 'Soukromé';

  @override
  String get profile => 'Profil';

  @override
  String get public => 'Veřejné';

  @override
  String get public_access => 'Veřejný přístup';

  @override
  String get public_share_everyone =>
      'Každý, kdo má odkaz, může tuto trasu na internetu vidět';

  @override
  String get public_share_limited =>
      'Odkaz mohou otevřít pouze lidé s přístupem';

  @override
  String get public_transport => 'Veřejná doprava';

  @override
  String get radius => 'Poloměr';

  @override
  String get railway_station => 'Železniční stanice';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get read_more => 'Číst dále';

  @override
  String get ready_to_join => 'Připraven se připojit';

  @override
  String get recalculate_elevation_data => 'Přepočítat údaje o nadmořské výšce';

  @override
  String get recalculating_elevation_data_hint =>
      'Při přepočítání výškových dat budou stávající výšková data, pokud existují, vymazána a nahrazena daty z Valhally.';

  @override
  String get register => 'Registrace';

  @override
  String get remote_users_cannot_edit =>
      'Vzdálení uživatelé nemohou provádět úpravy';

  @override
  String get removed_trail_from => 'Trasa odstraněna z';

  @override
  String get removed_trails_from => 'Trasy odstraněné z';

  @override
  String get required => 'Vyžadováno';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Obnovit';

  @override
  String get reset_password => 'Obnovit heslo';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get reverse_direction => 'Obrátit směr';

  @override
  String get road => 'Silnice';

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
  String get route_point => 'Bod trasy';

  @override
  String get russian => 'Ruština';

  @override
  String get save => 'Uložit';

  @override
  String get save_list => 'Uložit seznam';

  @override
  String get save_track => 'Save track';

  @override
  String get save_trail => 'Uložit trasu';

  @override
  String get save_your_trail_first => 'Nejprve uložte svou trasu';

  @override
  String get search => 'Search';

  @override
  String get search_cities => 'Vyhledat města';

  @override
  String get search_for_trails_places => 'Vyhledat trasy, seznamy, místa';

  @override
  String get search_list => 'Search list';

  @override
  String get search_places => 'Vyhledat místa';

  @override
  String get search_trails => 'Vyhledat trasy';

  @override
  String get select_date => 'Select date';

  @override
  String get select_list => 'Vybrat seznam';

  @override
  String get selected => 'vybráno';

  @override
  String get send_to => 'Send to...';

  @override
  String get set_private => 'Set private';

  @override
  String get set_public => 'Set public';

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
  String get settings_saved => 'Nastavení uloženo';

  @override
  String get share => 'Sdílet';

  @override
  String get share_profile => 'Sdílet profil';

  @override
  String get share_this_list => 'Sdílet tento seznam';

  @override
  String get share_this_trail => 'Sdílet tuto trasu';

  @override
  String get shared => 'Sdíleno';

  @override
  String get shared_by => 'Sdílel/a';

  @override
  String get shared_with => 'Sdíleno s';

  @override
  String get shelter => 'Přístřešek';

  @override
  String get shortest => 'nejkratší';

  @override
  String get show_in_overview => 'Zobrazit v přehledu';

  @override
  String get show_less => 'Zobrazit méně';

  @override
  String get show_on_map => 'Zobrazit na mapě';

  @override
  String get shower => 'Sprcha';

  @override
  String get skiing => 'Lyžování';

  @override
  String get slogan => 'Uložte si svá dobrodružství!';

  @override
  String get slope => 'Sklon';

  @override
  String get someone => 'Někdo';

  @override
  String get sort => 'Seřadit';

  @override
  String get spanish => 'Španělština';

  @override
  String get speed => 'Rychlost';

  @override
  String get start => 'Start';

  @override
  String get statistics => 'Statistiky';

  @override
  String get stop_drawing => 'Ukončit kreslení';

  @override
  String get stop_editing => 'Ukončit úpravy';

  @override
  String get strava_integration_after_date_hint =>
      'Pokud váš účet obsahuje velké množství aktivit, můžete narazit na limit API služby Strava, což znemožní synchronizaci všech aktivit najednou. Tomuto problému předejdete nastavením data \"Od\" a dále - synchronizují se tak pouze aktivity zaznamenané po tomto datu.';

  @override
  String get subcategories => 'Podkategorie';

  @override
  String get subway_stop => 'Vstup do metra';

  @override
  String get summit => 'Vrchol';

  @override
  String get summit_book => 'Vrcholová kniha';

  @override
  String get table => 'Stůl';

  @override
  String get tags => 'Štítky';

  @override
  String get text => 'Text';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get tilesets => 'Vlastní mapové vrstvy';

  @override
  String get time => 'Time';

  @override
  String get toilets => 'Toalety';

  @override
  String get top_speed => 'Nejvyšší rychlost';

  @override
  String get tourism => 'Turismus';

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
  String get trail_copied_successfully => 'trail copied successfully';

  @override
  String get trail_has_no_gpx => 'This trail has no GPX data.';

  @override
  String get trail_not_in_list => 'Trail is not in any list';

  @override
  String get trail_not_shared => 'Není sdíleno s nikým';

  @override
  String get trail_saved_successfully => 'Trasa úspěšně uložena';

  @override
  String get some_waypoints_failed_to_save =>
      'Trail saved, but some waypoints failed to save';

  @override
  String get trails_for_you => 'Trasy pro vás';

  @override
  String get tram_stop => 'Zastávka tramvaje';

  @override
  String get unchanged => 'beze změny';

  @override
  String get units => 'Jednotky';

  @override
  String get unlink => 'Zrušit propojení';

  @override
  String get upload_file => 'Nahrát soubor';

  @override
  String get upload_gpx => 'Nahrát GPX';

  @override
  String get upload_new_file => 'Nahrát nový soubor';

  @override
  String get uploaded => 'nahráno';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Successfully uploaded trail to Hammerhead';

  @override
  String get use_hills => 'Použít kopce';

  @override
  String get use_roads => 'Použít silnice';

  @override
  String get users => 'Users';

  @override
  String get username => 'Uživatelské jméno';

  @override
  String get username_not_unique =>
      'This username is already taken. Please try another.';

  @override
  String get view => 'Zobrazit';

  @override
  String get viewpoint => 'Vyhlídka';

  @override
  String get visibilty => 'Visibility';

  @override
  String get visibilty_status => 'Stav viditelnosti';

  @override
  String get walking_speed => 'Rychlost chůze';

  @override
  String get water => 'Voda';

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
  String get width => 'Šířka';

  @override
  String get wrong_username_or_password =>
      'Nesprávné uživatelské jméno nebo heslo';

  @override
  String get you_can => 'Můžete';

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
