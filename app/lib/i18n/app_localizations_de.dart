// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for German (`de`).
class AppLocalizationsDe extends AppLocalizations {
  AppLocalizationsDe([String locale = 'de']) : super(locale);

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
  String get add_bio => 'Bio hinzufügen';

  @override
  String get add_waypoint => 'Wegpunkt hinzufügen';

  @override
  String get after => 'Nach';

  @override
  String get all => 'Alle';

  @override
  String get altitude => 'Höhe';

  @override
  String get author => 'Autor';

  @override
  String get avatar => 'Avatar';

  @override
  String get average_speed => 'Durchschn. Geschwindigkeit';

  @override
  String get basic_info => 'Basisinformation';

  @override
  String get before => 'Vor';

  @override
  String get behavior => 'Verhalten';

  @override
  String get by => 'von';

  @override
  String get cancel => 'Abbrechen';

  @override
  String get car => 'Auto';

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
  String get change_email => 'Email ändern';

  @override
  String get change_password => 'Passwort ändern';

  @override
  String get clear_all => 'Alle ausblenden';

  @override
  String get close => 'Schließen';

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
  String get completion_status => 'Abschlussstatus';

  @override
  String get confirm_deletion => 'Löschen bestätigen';

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
  String get danger_zone => 'Gefahrenzone';

  @override
  String get date => 'Datum';

  @override
  String get delete => 'Löschen';

  @override
  String get open => 'Öffnen';

  @override
  String get delete_account => 'Konto löschen';

  @override
  String get delete_trail_confirm =>
      'Möchtest Du diese Route wirklich löschen? Diese Aktion kann nicht rückgängig gemacht werden.';

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
  String get distance => 'Distanz';

  @override
  String get download => 'Herunterladen';

  @override
  String get duration => 'Dauer';

  @override
  String get easy => 'Einfach';

  @override
  String get edit => 'Bearbeiten';

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
  String get error_reading_file => 'Fehler beim Lesen der Datei';

  @override
  String get error_saving_settings => 'Fehler beim Speichern der Einstellungen';

  @override
  String get error_saving_trail => 'Fehler beim Speichern der Route';

  @override
  String get error_updating_password =>
      'Fehler beim Aktualisieren des Passworts';

  @override
  String get explore => 'Erkunden';

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
  String get ferry => 'Fähre';

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
  String get follow => 'Folgen';

  @override
  String get followers => 'Follower';

  @override
  String get following => 'Folgt';

  @override
  String get from_photos => 'Aus Fotos';

  @override
  String get heading => 'Überschrift';

  @override
  String get height => 'Höhe';

  @override
  String get help => 'Hilfe';

  @override
  String get hiking => 'Wandern';

  @override
  String get home => 'Zuhause';

  @override
  String get hotel => 'Hotel';

  @override
  String get icon => 'Icon';

  @override
  String get imperial => 'Imperial';

  @override
  String get joined => 'Beigetreten';

  @override
  String get language => 'Sprache';

  @override
  String get latitude => 'Breitengrad';

  @override
  String get like_status => '\"Gefällt mir\" Status';

  @override
  String get liked => 'Gefällt mir';

  @override
  String get likes => 'Likes';

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
  String get logout => 'Logout';

  @override
  String get longitude => 'Längengrad';

  @override
  String get loop => 'Rundweg';

  @override
  String get map => 'Karte';

  @override
  String get metric => 'Metrisch';

  @override
  String get moderate => 'Mittel';

  @override
  String get more => 'weitere';

  @override
  String get mountain => 'Berg';

  @override
  String get my_account => 'Mein Konto';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String get name => 'Name';

  @override
  String get navigate => 'Navigieren';

  @override
  String get near => 'Nahe';

  @override
  String get new_password => 'Neues Passwort';

  @override
  String get new_password_success => 'Neues Passwort wurde gesetzt';

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
  String get no_comments_so_far => 'Bisher keine Kommentare';

  @override
  String get no_data => 'Keine Daten';

  @override
  String get no_description_for_now => 'Noch keine Beschreibung';

  @override
  String get no_gps_data_in_image => 'Keine GPS Daten im Bild';

  @override
  String get no_preference => 'Keine Präferenz';

  @override
  String get no_trails_found => 'No trails found';

  @override
  String get not_completed => 'Nicht abgeschlossen';

  @override
  String get notifications => 'Benachrichtigungen';

  @override
  String get only_me => 'Nur ich';

  @override
  String get or => 'oder';

  @override
  String get orientation => 'Ausrichtung';

  @override
  String get paragraph => 'Abschnitt';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Passwort';

  @override
  String get password_confirm => 'Passwort bestätigen';

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
  String get radius => 'Radius';

  @override
  String get reached_end_of_trail => 'Du hast das Ende des Wegs erreicht.';

  @override
  String get register => 'Registrieren';

  @override
  String get required => 'Pflichtfeld';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Zurücksetzen';

  @override
  String get resume => 'Fortsetzen';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => 'Aufnahme fortsetzen?';

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
  String get save => 'Speichern';

  @override
  String get save_recording_options => 'Save recording';

  @override
  String get save_track => 'Save track';

  @override
  String get search => 'Suche';

  @override
  String get search_for_trails_places => 'Suche nach Routen, Listen, Orten';

  @override
  String get select_date => 'Datum auswählen';

  @override
  String get selected => 'ausgewählt';

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
  String get share => 'Teilen';

  @override
  String get share_profile => 'Profil teilen';

  @override
  String get shared => 'Geteilt';

  @override
  String get show_on_map => 'Auf der Karte anzeigen';

  @override
  String get shower => 'Dusche';

  @override
  String get slogan => 'Speichere deine Abenteuer!';

  @override
  String get sort => 'Sortieren';

  @override
  String get speed => 'Geschwindigkeit';

  @override
  String get start => 'Start';

  @override
  String get subcategories => 'Unterkategorien';

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
  String get time => 'Zeit';

  @override
  String get time_in_motion => 'Time in Motion';

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
  String get trail_saved_successfully => 'Route gespeichert';

  @override
  String get some_waypoints_failed_to_save =>
      'Route gespeichert, aber einige Wegpunkte konnten nicht gespeichert werden';

  @override
  String get units => 'Einheiten';

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
  String get visibilty_status => 'Sichtbarkeit';

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

  @override
  String get something_went_wrong => 'Something went wrong';

  @override
  String get technical_details => 'Technical Details';

  @override
  String get link => 'Verknüpfen';

  @override
  String get url => 'URL';

  @override
  String get open_in_new_tab => 'In neuem Tab öffnen';

  @override
  String get remove => 'Entfernen';

  @override
  String get apply => 'Anwenden';

  @override
  String get add_at_least_2_anchors_hint =>
      'Add at least 2 anchors to see the elevation profile.';

  @override
  String get reverse_direction => 'Richtung umkehren';

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
  String get offline => 'Offline';

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
  String get edit_route => 'Route bearbeiten';

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
}
