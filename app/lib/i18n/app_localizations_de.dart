// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for German (`de`).
class AppLocalizationsDe extends AppLocalizations {
  AppLocalizationsDe([String locale = 'de']) : super(locale);

  @override
  String get biking => 'Radfahren';

  @override
  String get canoeing => 'Kanufahren';

  @override
  String get walking => 'Laufen';

  @override
  String get about => 'Über';

  @override
  String get appearance => 'Darstellung';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Du bist dabei, dein Konto zu löschen. Alle deine Routen werden ebenfalls gelöscht. Möchtest du fortfahren?';

  @override
  String get account_privacy => 'Privatsphäre des Kontos';

  @override
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Aktivitäten',
      one: 'Aktivität',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => 'Bio hinzufügen';

  @override
  String get add_entry => 'Eintrag hinzufügen';

  @override
  String get add_to_list => 'Listen verwalten';

  @override
  String get add_waypoint => 'Wegpunkt hinzufügen';

  @override
  String get added_trail_to => 'Route hinzugefügt zu';

  @override
  String get added_trails_to => 'Routen hinzugefügt zu';

  @override
  String get after => 'Nach';

  @override
  String get all => 'Alle';

  @override
  String get all_activities => 'Alle Aktivitäten';

  @override
  String get allow_auto_geolocate =>
      'Beginne das Zeichnen einer neuen Route am aktuellen Standort';

  @override
  String get alphabetical => 'Alphabetisch';

  @override
  String get already_account => 'Du hast bereits ein Konto?';

  @override
  String get altitude => 'Höhe';

  @override
  String get amenity => '';

  @override
  String get api_documentation => 'API Dokumentation';

  @override
  String get api_tokens => '';

  @override
  String get api_tokens_hint => '';

  @override
  String get apply_user_settings => '';

  @override
  String get attraction => 'Sehenswürdigkeit';

  @override
  String get author => 'Autor';

  @override
  String get avatar => 'Avatar';

  @override
  String get average_speed => 'Durchschn. Geschwindigkeit';

  @override
  String get avoid_bad_surfaces => 'Vermeide schlechte Oberflächen';

  @override
  String get back => 'Zurück';

  @override
  String get back_to_login => 'Zurück zum Login';

  @override
  String get bakery => 'Bäckerei';

  @override
  String get barrier => 'Barriere';

  @override
  String get basic_info => 'Basisinformation';

  @override
  String get basque => 'Baskisch';

  @override
  String get before => 'Vor';

  @override
  String get behavior => 'Verhalten';

  @override
  String get bicycle_parking => 'Fahrrad_Parkplatz';

  @override
  String get bicycle_rental => 'Fahrradverleih';

  @override
  String get bicycle_shop => 'Fahrrad_Reparatur';

  @override
  String get bike_type => 'Fahrradtyp';

  @override
  String get bus_stop => 'Bushaltestelle';

  @override
  String get by => 'von';

  @override
  String get campsite => 'Campingplatz';

  @override
  String get can => 'kann';

  @override
  String get cancel => 'Abbrechen';

  @override
  String get car => 'Auto';

  @override
  String get car_motorcycle => 'Auto/Motorrad';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Karten',
      one: 'Karte',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Kategorien';

  @override
  String get category => 'Kategorie';

  @override
  String get change => 'Ändern';

  @override
  String get change_email => 'Email ändern';

  @override
  String get change_password => 'Passwort ändern';

  @override
  String get changelog => 'Änderungshistorie';

  @override
  String get chinese => 'Chinesisch (vereinfacht)';

  @override
  String get clear_all => 'Alle ausblenden';

  @override
  String get climbing => 'Klettern';

  @override
  String get close => 'Schließen';

  @override
  String get collapse_trail_list => '';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Kommentare',
      one: 'Kommentar',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Abgeschlossen';

  @override
  String get completed_a_trail => 'hat eine Route abgeschlossen';

  @override
  String get completed_tours => 'Abgeschlossene Touren';

  @override
  String get completion_status => 'Abschlussstatus';

  @override
  String get confirm => 'Bestätigen';

  @override
  String get confirm_deletion => 'Löschen bestätigen';

  @override
  String get confirm_publish => 'Veröffentlichung bestätigen';

  @override
  String get confirm_share => 'Teilen bestätigen';

  @override
  String get connect => 'Verbinden';

  @override
  String get contribute => 'Mitwirken';

  @override
  String get couldnt_start_navigation =>
      'Navigation konnte nicht gestartet werden. Prüfe deine Verbindung und versuche es erneut.';

  @override
  String get location_services_disabled =>
      'Ortungsdienste sind deaktiviert. Bitte aktiviere GPS, um die Navigation zu nutzen.';

  @override
  String get location_permission_denied =>
      'Für die Navigation wird eine Standortberechtigung benötigt.';

  @override
  String get location_permission_permanently_denied =>
      'Die Standortberechtigung wurde dauerhaft verweigert. Bitte aktiviere sie in den Einstellungen.';

  @override
  String get location_unavailable =>
      'Unable to determine your location. Please try again.';

  @override
  String get copy_link => 'Link kopieren';

  @override
  String get create_new_list => 'Neue Liste erstellen';

  @override
  String get create_waypoint => 'Wegpunkt erstellen';

  @override
  String get creation_date => 'Erstellungsdatum';

  @override
  String get crop => 'Zuschneiden';

  @override
  String get cross => 'Querfeldein';

  @override
  String get current_password => 'Aktuelles Passwort';

  @override
  String get cycling => 'Radfahren';

  @override
  String get cycling_speed => 'Radfahrgeschwindigkeit';

  @override
  String get czech => 'Tschechisch';

  @override
  String get danger_zone => 'Gefahrenzone';

  @override
  String get date => 'Datum';

  @override
  String get default_category => 'Standardkategorie';

  @override
  String get default_location => 'Standort';

  @override
  String get degrees => 'Grad';

  @override
  String get delete => 'Löschen';

  @override
  String get open => 'Öffnen';

  @override
  String get delete_account => 'Konto löschen';

  @override
  String get delete_list_confirm =>
      'Möchtest Du diese Liste wirklich löschen? Die Routen in der Liste sind danach weiterhin verfügbar.';

  @override
  String get delete_summit_log_confirm =>
      'Möchtest du diesen Gipfelbuch_Eintrag wirklich löschen? Diese Aktion kann nicht rückgängig gemacht werden.';

  @override
  String get delete_trail_confirm =>
      'Möchtest Du diese Route wirklich löschen? Diese Aktion kann nicht rückgängig gemacht werden.';

  @override
  String get describe_your_trail => 'Beschreibe deine Route';

  @override
  String get description => 'Beschreibung';

  @override
  String get difficult => 'Schwer';

  @override
  String get difficulty => 'Schwierigkeit';

  @override
  String get directions => 'Wegbeschreibung';

  @override
  String get display => 'Anzeigen';

  @override
  String get display_as => 'Anzeigen als';

  @override
  String get distance => 'Distanz';

  @override
  String get documentation => 'Dokumentation';

  @override
  String get download => 'Herunterladen';

  @override
  String get draw_a_route => 'Route zeichnen';

  @override
  String get driving => 'Auto';

  @override
  String get duplicate => 'Duplizieren';

  @override
  String get duration => 'Dauer';

  @override
  String get dutch => 'Niederländisch';

  @override
  String get easy => 'Einfach';

  @override
  String get edit => 'Bearbeiten';

  @override
  String get edit_entry => 'Eintrag bearbeiten';

  @override
  String get edit_list => 'Liste bearbeiten';

  @override
  String get edit_route => 'Route bearbeiten';

  @override
  String get edit_waypoint => 'Wegpunkt bearbeiten';

  @override
  String get edited => 'bearbeitet';

  @override
  String get elevation_gain => 'Höhenunterschied (aufw.)';

  @override
  String get elevation_loss => 'Höhenunterschied (abw.)';

  @override
  String get elevation_profile => 'Höhenprofil';

  @override
  String get email => 'Email';

  @override
  String get email_not_unique => 'Diese Email_Adresse wird bereits verwendet';

  @override
  String get email_updated => 'E_Mail aktualisiert';

  @override
  String get email_verified => 'Email verifiziert';

  @override
  String empty_activities(Object username) {
    return '$username hat noch keine Aktivitäten';
  }

  @override
  String empty_bio(Object username) {
    return '$username hat noch keine Biografie hinzugefügt';
  }

  @override
  String get empty_feed => 'Dein Feed ist leer';

  @override
  String get empty_feed_explanation =>
      'Aktivitäten von dir oder Personen, denen du folgst, werden hier angezeigt';

  @override
  String empty_lists(Object username) {
    return '$username hat keine öffentlichen Listen';
  }

  @override
  String get enable_auto_routing => 'Auto_Routing aktivieren';

  @override
  String get english => 'Englisch';

  @override
  String get entry => 'Eintrag';

  @override
  String get error_copying_trail => 'Fehler beim Kopieren der Route';

  @override
  String get error_creating_user => 'Fehler beim Erstellen des Nutzers';

  @override
  String get error_deleting_token => '';

  @override
  String get error_disabling_strava_integration =>
      'Fehler beim Deaktivieren der Stravaintegration';

  @override
  String get error_during_login => 'Fehler beim Login';

  @override
  String get error_during_password_reset =>
      'Email zum Zurücksetzen des Passworts konnte nicht versandt werden';

  @override
  String get error_exporting_trail => 'Fehler beim Exportieren der Route';

  @override
  String get error_generating_token => '';

  @override
  String get error_liking_trail => 'Error liking trail';

  @override
  String get error_logging_in_to_hammerhead =>
      'Fehler bei der Anmeldung bei Hammerhead';

  @override
  String get error_logging_in_to_komoot =>
      'Fehler bei der Anmeldung bei komoot';

  @override
  String get error_posting_comment => 'Fehler beim Posten des Kommentars';

  @override
  String get error_printing_map => 'Fehler beim Drucken der Karte';

  @override
  String get error_reading_file => 'Fehler beim Lesen der Datei';

  @override
  String get error_saving_list => 'Fehler beim Speichern der Liste';

  @override
  String get error_saving_settings => 'Fehler beim Speichern der Einstellungen';

  @override
  String get error_saving_trail => 'Fehler beim Speichern der Route';

  @override
  String error_setting_up_integration(Object provider) {
    return 'Fehler beim Einrichten der ${provider}_Integration';
  }

  @override
  String get error_updating_hammerhead_integration =>
      'Fehler bei Aktualisierung der Hammerhead_Integration';

  @override
  String get error_updating_komoot_integration =>
      'Fehler bei Aktualisierung der komoot_Integration';

  @override
  String get error_updating_password =>
      'Fehler beim Aktualisieren des Passworts';

  @override
  String get error_updating_strava_integration =>
      'Fehler bei Aktualisierung der Strava_Integration';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Fehler beim Hochladen der Route zu Hammerhead';

  @override
  String get est_duration => 'Gesch. Dauer';

  @override
  String get everyone_with_the_link => 'Jeder mit dem Link';

  @override
  String get expand_trail_list => '';

  @override
  String get expiration => '';

  @override
  String get expires => '';

  @override
  String get explore => 'Erkunden';

  @override
  String get explore_some_trails => 'Erkunde einige Routen';

  @override
  String get exit_navigation => 'Beenden';

  @override
  String get stop_navigation_confirm =>
      'Navigation beenden und zur Route zurückkehren?';

  @override
  String get stop_recording => 'Aufnahme stoppen';

  @override
  String get stop_recording_confirm => 'Aufnahme stoppen?';

  @override
  String get search_this_area => 'In diesem Bereich suchen';

  @override
  String get export => 'Exportieren';

  @override
  String get export_all_trails => 'Alle Routen exportieren';

  @override
  String get favourite_sport => 'Lieblingssportart';

  @override
  String get features => 'Features';

  @override
  String get ferry => 'Fähre';

  @override
  String get file_format => 'Dateiformat';

  @override
  String file_too_big(Object file, Object size) {
    return 'Datei $file ist zu groß (max. $size)';
  }

  @override
  String get filter_categories => 'Kategorien filtern';

  @override
  String get filter_difficulty => 'Schwierigkeit filtern';

  @override
  String get filter_tags => 'Tags filtern';

  @override
  String get filter_trails => 'Routen filtern';

  @override
  String get finish => 'Ziel';

  @override
  String get finish_disabled_hint =>
      'Add at least 2 anchors to finish your route.';

  @override
  String get fixed_speed => 'Fixe Geschwindigkeit';

  @override
  String get focus_map_on => 'Karte fokussieren auf';

  @override
  String get follow => 'Folgen';

  @override
  String get follow_request_pending => 'Anfrage ausstehend';

  @override
  String get followers => 'Follower';

  @override
  String get following => 'Folgt';

  @override
  String get food => 'Essen';

  @override
  String get food_drinks => 'Essen & Trinken';

  @override
  String get forgot_your_password => 'Passwort vergessen?';

  @override
  String get french => 'Französisch';

  @override
  String get from_file => 'Aus einer Datei';

  @override
  String get from_photos => 'Aus Fotos';

  @override
  String get from_url => 'Aus einer URL';

  @override
  String get garage => 'Auto Reparatur';

  @override
  String get gas_station => 'Tankstelle';

  @override
  String get generate_new_token => '';

  @override
  String get german => 'Deutsch';

  @override
  String get get_position_from_exif => 'Koordinaten aus EXIF Daten';

  @override
  String get get_started => 'Los geht’s';

  @override
  String get grid => 'Gitter';

  @override
  String get grocery_store => 'Lebensmittelgeschäft';

  @override
  String get hammerhead_integration_after_date_hint =>
      'Wenn Ihr Hammerhead Konto bereits mit anderen Trail_Datenbanken wie komoot oder Strava synchronisiert ist, kann die zusätzliche Synchronisierung Ihrer Hammerhead_Daten zu Duplikaten führen. Um dies zu vermeiden, können Sie unten ein Startdatum festlegen, sodass nur Aktivitäten synchronisiert werden, die nach diesem Datum aufgezeichnet wurden.';

  @override
  String get heading => 'Überschrift';

  @override
  String get height => 'Höhe';

  @override
  String get help => 'Hilfe';

  @override
  String get hero_section_0_text =>
      'Entdecke spannende Routen, speichere deine Favoriten und erlebe die Schönheit der Natur. Finde dein nächstes Abenteuer!';

  @override
  String get hero_section_1_heading => 'Hier gibt es noch keine Routen.';

  @override
  String get hero_section_1_text =>
      'Hier sind einige Routen, die dir gefallen könnten. Oder du wirfst einen Blick auf die vollständige Liste.';

  @override
  String get hero_section_1_text_alternative =>
      'Speichere dein letztes Abenteuer, um loszulegen.';

  @override
  String get hero_section_2_text =>
      'Wusstest du schon? Du kannst nicht nur deine Wanderrouten speichern. Es gibt viele Kategorien für alle deine Abenteuer.';

  @override
  String get hiking => 'Wandern';

  @override
  String get home => 'Zuhause';

  @override
  String get hotel => 'Hotel';

  @override
  String get hungarian => 'Ungarisch';

  @override
  String get hut => 'Hütte';

  @override
  String get hybrid => 'Hybrid';

  @override
  String get icon => 'Icon';

  @override
  String get ignore_trails_before_date => 'Routen vor diesem Datum ignorieren';

  @override
  String get imperial => 'Imperial';

  @override
  String get import => 'Importieren';

  @override
  String get import_hint =>
      'GPX, FIT, KML oder TCX Dateien auswählen oder hierher ziehen...';

  @override
  String get include_description => 'Beschreibung übernehmen';

  @override
  String get include_waypoints => 'Wegpunkte einbeziehen';

  @override
  String get integration_description_hammerhead =>
      'Synchronisiert Deine Hammerhead_Touren regelmäßig mit wanderer.';

  @override
  String get integration_description_komoot =>
      'Synchronisiert Deine komoot_Touren regelmäßig mit wanderer.';

  @override
  String get integration_description_strava =>
      'Synchronisiert Deine Strava_Routen und _Aktivitäten regelmäßig mit wanderer.';

  @override
  String get integration_disabled => 'Integration deaktiviert';

  @override
  String get integration_enabled => 'Integration aktiviert';

  @override
  String get integration_privacy_hint_original => '';

  @override
  String get integration_privacy_hint_user => '';

  @override
  String get integrations => 'Integrationen';

  @override
  String get invalid_date => 'Ungültiges Datum';

  @override
  String get invalid_username => 'Ungültiger Nutzername';

  @override
  String get italian => 'Italienisch';

  @override
  String get joined => 'Beigetreten';

  @override
  String get keep_original => '';

  @override
  String get keep_private => 'Ohne Veröffentlichung fortfahren';

  @override
  String get language => 'Sprache';

  @override
  String get last_used => '';

  @override
  String get latitude => 'Breitengrad';

  @override
  String layer(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Ebenen',
      one: 'Ebene',
    );
    return '$_temp0';
  }

  @override
  String get license => 'Lizenz';

  @override
  String get like_status => '\"Gefällt mir\" Status';

  @override
  String get liked => 'Gefällt mir';

  @override
  String get likes => 'Likes';

  @override
  String get limited => 'Begrenzt';

  @override
  String get link_copied => 'Link kopiert!';

  @override
  String get linked_lists => 'Verknüpfte Listen';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Listen',
      one: 'Liste',
    );
    return '$_temp0';
  }

  @override
  String get list_not_shared => 'Mit niemandem geteilt';

  @override
  String get list_public_warning =>
      'Alle Routen in dieser Liste werden veröffentlicht.';

  @override
  String get list_saved_successfully => 'Liste gespeichert';

  @override
  String get list_share_warning =>
      'Durch das Teilen einer Liste werden automatisch alle darin enthaltenen Routen freigegeben.';

  @override
  String get list_share_warning_update =>
      'Hinzugefügte Routen werden mit allen geteilt, die Zugriff auf diese Liste haben.';

  @override
  String get location => 'Standort';

  @override
  String get locations => 'Orte';

  @override
  String get center_on_my_location => 'Center on my location';

  @override
  String get location_tracking_notification_title => 'Wanderer';

  @override
  String get location_tracking_notification_text => 'Recording your trail';

  @override
  String get login => 'Login';

  @override
  String get login_details => 'Anmeldedaten';

  @override
  String get logout => 'Logout';

  @override
  String get longitude => 'Längengrad';

  @override
  String get loop => 'Rundweg';

  @override
  String get make_one => 'Neues erstellen!';

  @override
  String get make_thumbnail => 'Thumbnail festlegen';

  @override
  String get map => 'Karte';

  @override
  String get map_style => 'Kartenstil';

  @override
  String get max_hiking_difficulty => 'Max. Schwierigkeit der Wanderung';

  @override
  String get metric => 'Metrisch';

  @override
  String get moderate => 'Mittel';

  @override
  String get more => 'weitere';

  @override
  String get more_route_settings => 'Weitere Routen_Einstellungen';

  @override
  String get mountain => 'Berg';

  @override
  String get mountain_pass => 'Bergpass';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return 'Muss mindestens $n Zeichen lang sein';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return 'Darf höchstens $n Zeichen lang sein';
  }

  @override
  String get my_account => 'Mein Konto';

  @override
  String get my_trails => 'Meine Routen';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return 'vor $n Tagen';
  }

  @override
  String n_hours_ago(Object n) {
    return 'vor $n Stunden';
  }

  @override
  String n_minutes_ago(Object n) {
    return 'vor $n Minuten';
  }

  @override
  String n_months_ago(Object n) {
    return 'vor $n Monaten';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'vor $n Sekunden',
      zero: 'gerade eben',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return 'vor $n Jahren';
  }

  @override
  String get name => 'Name';

  @override
  String get navigate => 'Navigieren';

  @override
  String get near => 'Nahe';

  @override
  String get never => '';

  @override
  String get new_list => 'Neue Liste';

  @override
  String get new_password => 'Neues Passwort';

  @override
  String get new_password_error => 'Fehler beim Speichern des Passworts';

  @override
  String get new_password_success => 'Neues Passwort wurde gesetzt';

  @override
  String get new_password_text => 'Wähle ein neues Passwort';

  @override
  String get new_token_generated => '';

  @override
  String get new_trail => 'Neue Route';

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
  String get no_account => 'Du hast noch kein Konto?';

  @override
  String get no_api_tokens => '';

  @override
  String get no_comments_so_far => 'Bisher keine Kommentare';

  @override
  String get no_data => 'Keine Daten';

  @override
  String get no_description_for_now => 'Noch keine Beschreibung';

  @override
  String get no_gps_data_in_image => 'Keine GPS Daten im Bild';

  @override
  String get no_grid => 'Kein Gitter';

  @override
  String get no_notifications => 'Keine Benachrichtigungen';

  @override
  String get no_photos_here => 'Hier sind noch keine Fotos';

  @override
  String get no_preference => 'Keine Präferenz';

  @override
  String get no_results => 'Keine Ergebnisse gefunden';

  @override
  String get no_routes_added => 'Keine Routen hinzugefügt';

  @override
  String get no_trails_found => 'No trails found';

  @override
  String get no_waypoints_yet => 'Noch keine Wegpunkte';

  @override
  String get norwegian => 'Norwegisch';

  @override
  String get not_a_valid_email_address => 'Keine gültige Email_Adresse';

  @override
  String get not_a_valid_url => 'Keine gültige URL';

  @override
  String get not_completed => 'Nicht abgeschlossen';

  @override
  String notification_comment_mention(Object user) {
    return '$user hat Dich in einem Kommentar erwähnt';
  }

  @override
  String notification_list_create(Object user) {
    return '$user hat eine neue Liste erstellt';
  }

  @override
  String notification_list_share(Object user) {
    return '$user hat eine Liste mit Dir geteilt';
  }

  @override
  String get notification_new_follower => 'Du hast einen neuen Follower';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user hat einen Gipfelbuch_Eintrag zu deiner Route „$trail“ erstellt';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user hat Dich in einem Gipfelbuch_Eintrag erwähnt';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user hat einen Kommentar zu Deiner Route \"$trail\" hinterlassen';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user hat eine neue Route erstellt';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user hat deinem Trail $trail ein „Gefällt mir“ gegeben';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user hat dich in seiner Route erwähnt';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user hat eine Route mit Dir geteilt';
  }

  @override
  String get notifications => 'Benachrichtigungen';

  @override
  String object_share_error(Object object) {
    return 'Eine $object muss öffentlich sein, um über Instanzen hinweg geteilt zu werden.';
  }

  @override
  String get off => 'Aus';

  @override
  String get only_me => 'Nur ich';

  @override
  String get open_in_new_tab => 'In neuem Tab öffnen';

  @override
  String get or => 'oder';

  @override
  String get orientation => 'Ausrichtung';

  @override
  String get paper_size => 'Papierformat';

  @override
  String get paragraph => 'Abschnitt';

  @override
  String get parking => 'Parken';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Passwort';

  @override
  String get password_confirm => 'Passwort bestätigen';

  @override
  String get password_reset_sent =>
      'Email zum Zurücksetzen des Passworts versandt';

  @override
  String get password_reset_text =>
      'Wir senden Dir einen Link zum Zurücksetzen Deines Passworts zu';

  @override
  String get password_updated => 'Passwort aktualisiert';

  @override
  String get passwords_must_match => 'Passwörter müssen übereinstimmen';

  @override
  String get photos => 'Fotos';

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
  String get pick_a_trail => 'Route auswählen';

  @override
  String get planned_a_trail => 'hat eine Route geplant';

  @override
  String get planned_tours => 'Geplante Touren';

  @override
  String get pois => 'Interessante Orte';

  @override
  String get polish => 'Polnisch';

  @override
  String get portuguese => 'Portugiesisch';

  @override
  String get print => 'Drucken';

  @override
  String get privacy => 'Privatsphäre';

  @override
  String get private => 'Privat';

  @override
  String get profile => 'Profil';

  @override
  String get public => 'Öffentlich';

  @override
  String get public_access => 'Öffentlicher Zugriff';

  @override
  String get public_share_everyone =>
      'Jeder im Internet mit dem Link kann diese Route sehen';

  @override
  String get public_share_limited =>
      'Nur Leute mit Zugriff können diese Route sehen';

  @override
  String get public_transport => 'Öffentlicher Verkehr';

  @override
  String get radius => 'Radius';

  @override
  String get railway_station => 'Bahnhof';

  @override
  String get reached_end_of_trail => 'Du hast das Ende des Wegs erreicht.';

  @override
  String get read_more => 'Mehr';

  @override
  String get ready_to_join => 'Bereit loszulegen';

  @override
  String get recalculate_elevation_data => 'Höhendaten neu berechnen';

  @override
  String get recalculating_elevation_data_hint =>
      'Dies löscht existierende Höhendaten, falls vorhanden, und ersetzt sie mit Daten aus Valhalla.';

  @override
  String get register => 'Registrieren';

  @override
  String get remote_users_cannot_edit =>
      'Remote_Benutzer können nicht bearbeiten';

  @override
  String get removed_trail_from => 'Route entfernt aus';

  @override
  String get removed_trails_from => 'Routen entfernt aus';

  @override
  String get required => 'Pflichtfeld';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Zurücksetzen';

  @override
  String get reset_password => 'Passwort zurücksetzen';

  @override
  String get resume => 'Fortsetzen';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => 'Aufnahme fortsetzen?';

  @override
  String get reverse_direction => 'Richtung umkehren';

  @override
  String get road => 'Straße';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Routen',
      one: 'Route',
    );
    return '$_temp0';
  }

  @override
  String get route_point => 'Punkt auf Route';

  @override
  String get russian => 'Russisch';

  @override
  String get save => 'Speichern';

  @override
  String get save_list => 'Liste speichern';

  @override
  String get save_track => 'Save track';

  @override
  String get save_trail => 'Route speichern';

  @override
  String get save_your_trail_first => 'Route zuerst speichern';

  @override
  String get search => 'Suche';

  @override
  String get search_cities => 'Städte suchen';

  @override
  String get search_for_trails_places => 'Suche nach Routen, Listen, Orten';

  @override
  String get search_list => 'Liste suchen';

  @override
  String get search_places => 'Orte suchen';

  @override
  String get search_trails => 'Route suchen';

  @override
  String get select_date => 'Datum auswählen';

  @override
  String get select_list => 'Liste auswählen';

  @override
  String get selected => 'ausgewählt';

  @override
  String get send_to => 'Senden an...';

  @override
  String get set_private => 'Verbergen';

  @override
  String get set_public => 'Veröffentlichen';

  @override
  String get settings => 'Einstellungen';

  @override
  String get settings_notification_comment_mention =>
      'Jemand hat dich in einem Kommentar erwähnt';

  @override
  String get settings_notification_list_share =>
      'Jemand hat eine Liste mit Dir geteilt';

  @override
  String get settings_notification_new_follower =>
      'Du hast einen neuen Follower';

  @override
  String get settings_notification_summit_log_create =>
      'Jemand hat einen Gipfelbuch_Eintrag zu deiner Route erstellt';

  @override
  String get settings_notification_summit_log_mention =>
      'Jemand hat dich in einem Gipfelbuch Eintrag erwähnt';

  @override
  String get settings_notification_trail_comment =>
      'Jemand hat einen Kommentar zu Deiner Route hinterlassen';

  @override
  String get settings_notification_trail_like =>
      'Jemand hat deiner Route ein \"Gefällt mir\" gegeben';

  @override
  String get settings_notification_trail_mention =>
      'Jemand hat dich in einer Route erwähnt';

  @override
  String get settings_notification_trail_share =>
      'Jemand hat eine Route mit Dir geteilt';

  @override
  String get settings_privacy_account_private =>
      'Nur Du kannst dein Profil sehen. Du wirst nicht in den Suchergebnissen angezeigt. Andere Benutzer können Dir nicht folgen oder Routen mit Dir teilen. Du kannst weiterhin Routen oder Listen veröffentlichen.';

  @override
  String get settings_privacy_account_public =>
      'Jeder kann Dein Profil sehen. Du erscheinst in den Suchergebnissen. Andere Benutzer können Dir folgen und Routen mit Dir teilen.';

  @override
  String get settings_privacy_lists_private =>
      'Deine Listen sind standardmäßig privat. Niemand außer Dir kann sie sehen. Du kannst diese Einstellung jederzeit für einzelne Listen ändern.';

  @override
  String get settings_privacy_lists_public =>
      'Deine Listen sind standardmäßig öffentlich. Jeder kann sie sehen. Du kannst diese Einstellung jederzeit für einzelne Listen ändern.';

  @override
  String get settings_privacy_trails_private =>
      'Deine Routen sind standardmäßig privat. Niemand außer Dir kann sie sehen. Du kannst diese Einstellung jederzeit für einzelne Routen ändern.';

  @override
  String get settings_privacy_trails_public =>
      'Deine Trails sind standardmäßig öffentlich. Jeder kann sie sehen. Du kannst diese Einstellung jederzeit für einzelne Routen ändern.';

  @override
  String get settings_saved => 'Einstellungen gespeichert';

  @override
  String get share => 'Teilen';

  @override
  String get share_profile => 'Profil teilen';

  @override
  String get share_this_list => 'Diese Liste teilen';

  @override
  String get share_this_trail => 'Diese Route teilen';

  @override
  String get shared => 'Geteilt';

  @override
  String get shared_by => 'Geteilt von';

  @override
  String get shared_with => 'Geteilt mit';

  @override
  String get shelter => 'Schutzhütte';

  @override
  String get shortest => 'kürzeste';

  @override
  String get show_in_overview => 'In der Übersicht anzeigen';

  @override
  String get show_less => 'Weniger anzeigen';

  @override
  String get show_on_map => 'Auf der Karte anzeigen';

  @override
  String get shower => 'Dusche';

  @override
  String get skiing => 'Skifahren';

  @override
  String get slogan => 'Speichere deine Abenteuer!';

  @override
  String get slope => 'Steigung';

  @override
  String get someone => 'Jemand';

  @override
  String get sort => 'Sortieren';

  @override
  String get spanish => 'Spanisch';

  @override
  String get speed => 'Geschwindigkeit';

  @override
  String get start => 'Start';

  @override
  String get statistics => 'Statistiken';

  @override
  String get stop_drawing => 'Zeichnen beenden';

  @override
  String get stop_editing => 'Bearbeiten beenden';

  @override
  String get strava_integration_after_date_hint =>
      'Wenn Ihr Konto eine große Anzahl von Aktivitäten enthält, kann es vorkommen, dass Sie aufgrund der API_Restriktionen von Strava nicht alle Aktivitäten auf einmal synchronisieren können. Um dieses Problem zu umgehen, können Sie unten ein Startdatum festlegen, sodass nur Aktivitäten synchronisiert werden, die nach diesem Datum aufgezeichnet wurden.';

  @override
  String get subcategories => 'Unterkategorien';

  @override
  String get subway_stop => 'U_Bahn Eingang';

  @override
  String get summit => 'Gipfel';

  @override
  String get summit_book => 'Gipfelbuch';

  @override
  String get table => 'Tabelle';

  @override
  String get tags => 'Tags';

  @override
  String get text => 'Text';

  @override
  String get theme_dark => 'Dunkel';

  @override
  String get theme_light => 'Hell';

  @override
  String get theme_system => 'System folgen';

  @override
  String get tilesets => 'Tilesets';

  @override
  String get time => 'Zeit';

  @override
  String get time_in_motion => 'Time in Motion';

  @override
  String get toilets => 'Toiletten';

  @override
  String get top_speed => 'Höchstgeschwindigkeit';

  @override
  String get tourism => 'Tourismus';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Routen',
      one: 'Route',
    );
    return '$_temp0';
  }

  @override
  String get trail_copied_successfully => 'Route erfolgreich kopiert';

  @override
  String get trail_has_no_gpx => 'Diese Route besitzt keine GPX Daten.';

  @override
  String get trail_not_in_list => 'Trail gehört zu keiner Liste.';

  @override
  String get trail_not_shared => 'Mit niemandem geteilt';

  @override
  String get trail_saved_successfully => 'Route gespeichert';

  @override
  String get some_waypoints_failed_to_save =>
      'Route gespeichert, aber einige Wegpunkte konnten nicht gespeichert werden';

  @override
  String get trails_for_you => 'Routen für dich';

  @override
  String get tram_stop => 'Tram Haltestelle';

  @override
  String get unchanged => 'unverändert';

  @override
  String get units => 'Einheiten';

  @override
  String get unlink => 'Trennen';

  @override
  String get upload_file => 'Datei hochladen';

  @override
  String get upload_gpx => 'GPX hochladen';

  @override
  String get upload_new_file => 'Neue Datei hochladen';

  @override
  String get uploaded => 'hochgeladen';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Route erfolgreich zu Hammerhead hochgeladen';

  @override
  String get use_hills => 'Hügel einbeziehen';

  @override
  String get use_roads => 'Nutze Straßen';

  @override
  String get users => 'Benutzer';

  @override
  String get username => 'Nutzername';

  @override
  String get username_not_unique =>
      'Dieser Nutzername ist bereits vergeben. Bitte versuche es mit einem anderen.';

  @override
  String get view => 'Ansehen';

  @override
  String get viewpoint => 'Aussichtspunkt';

  @override
  String get visibilty => '';

  @override
  String get visibilty_status => 'Sichtbarkeit';

  @override
  String get walking_speed => 'Laufgeschwindigkeit';

  @override
  String get water => 'Wasser';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Wegpunkte',
      one: 'Wegpunkt',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Willkommen bei';

  @override
  String get width => 'Breite';

  @override
  String get wrong_username_or_password =>
      'Falscher Nutzername oder falsches Passwort';

  @override
  String get you_can => 'Du kannst';

  @override
  String get you_have_arrived => 'Angekommen';

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
