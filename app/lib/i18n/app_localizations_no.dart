// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Norwegian (`no`).
class AppLocalizationsNo extends AppLocalizations {
  AppLocalizationsNo([String locale = 'no']) : super(locale);

  @override
  String get biking => 'Biking';

  @override
  String get canoeing => 'Canoeing';

  @override
  String get walking => 'Walking';

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
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Aktiviteter',
      one: 'Aktivitet',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => 'Legg til biografi';

  @override
  String get add_entry => 'Legg til oppføring';

  @override
  String get add_to_list => 'Legg til i liste';

  @override
  String get add_waypoint => 'Legg til veipunkt';

  @override
  String get added_trail_to => 'La til sti i';

  @override
  String get added_trails_to => 'La til stier i';

  @override
  String get after => 'Etter';

  @override
  String get all => 'All';

  @override
  String get all_activities => 'Alle aktiviteter';

  @override
  String get allow_auto_geolocate =>
      'Begynn å tegne en ny sti fra din nåværende posisjon';

  @override
  String get alphabetical => 'Alfabetisk';

  @override
  String get already_account => 'Har du allerede en konto?';

  @override
  String get altitude => 'Høyde';

  @override
  String get amenity => 'Fasilitet';

  @override
  String get api_documentation => 'API-dokumentasjon';

  @override
  String get api_tokens => 'API-tokens';

  @override
  String get api_tokens_hint =>
      'API-tokens kan brukes til å gi tredjepartsapplikasjoner tilgang til din Wanderer-konto.';

  @override
  String get apply_user_settings => 'Bruk brukerinnstillinger';

  @override
  String get attraction => 'Attraksjon';

  @override
  String get author => 'Forfatter';

  @override
  String get avatar => 'Avatar';

  @override
  String get average_speed => 'Gj.snittsfart';

  @override
  String get avoid_bad_surfaces => 'Unngå dårlig underlag';

  @override
  String get back => 'Tilbake';

  @override
  String get back_to_login => 'Tilbake til innlogging';

  @override
  String get bakery => 'Bakeri';

  @override
  String get barrier => 'Barriere';

  @override
  String get basic_info => 'Grunnleggende info';

  @override
  String get basque => 'Baskisk';

  @override
  String get before => 'Før';

  @override
  String get behavior => 'Oppførsel';

  @override
  String get bicycle_parking => 'Sykkelparkering';

  @override
  String get bicycle_rental => 'Sykkel-utleie';

  @override
  String get bicycle_shop => 'Sykkelbutikk';

  @override
  String get bike_type => 'Sykkeltype';

  @override
  String get bus_stop => 'Busstopp';

  @override
  String get by => 'av';

  @override
  String get campsite => 'Campingplass';

  @override
  String get can => 'kan';

  @override
  String get cancel => 'Avbryt';

  @override
  String get car => 'Bil';

  @override
  String get car_motorcycle => 'Bil/Motorsykkel';

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
  String get change => 'Endre';

  @override
  String get change_email => 'Endre e-post';

  @override
  String get change_password => 'Endre passord';

  @override
  String get changelog => 'Endringslogg';

  @override
  String get chinese => 'Kinesisk (forenklet)';

  @override
  String get clear_all => 'Fjern alle';

  @override
  String get climbing => 'Klatring';

  @override
  String get close => 'Lukk';

  @override
  String get collapse_trail_list => 'Skjul stiliste';

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
  String get completed_a_trail => 'fullførte en sti';

  @override
  String get completed_tours => 'Fullførte turer';

  @override
  String get completion_status => 'Fullføringsstatus';

  @override
  String get confirm => 'Bekreft';

  @override
  String get confirm_deletion => 'Bekreft sletting';

  @override
  String get confirm_publish => 'Bekreft publisering';

  @override
  String get confirm_share => 'Bekreft deling';

  @override
  String get connect => 'Koble til';

  @override
  String get contribute => 'Bidra';

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
  String get copy_link => 'Kopier lenke';

  @override
  String get create_new_list => 'Opprett ny liste';

  @override
  String get create_waypoint => 'Opprett veipunkt';

  @override
  String get creation_date => 'Opprettelsesdato';

  @override
  String get crop => 'Beskjær';

  @override
  String get cross => 'Kryss';

  @override
  String get current_password => 'Nåværende passord';

  @override
  String get cycling => 'Sykling';

  @override
  String get cycling_speed => 'Sykkelfart';

  @override
  String get czech => 'Tsjekkisk';

  @override
  String get danger_zone => 'Faresone';

  @override
  String get date => 'Dato';

  @override
  String get default_category => 'Standardkategori';

  @override
  String get default_location => 'Standardplassering';

  @override
  String get degrees => 'Grader';

  @override
  String get delete => 'Slett';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Slett konto';

  @override
  String get delete_list_confirm =>
      'Vil du virkelig slette denne listen? Stiene i listen vil fortsatt være tilgjengelige.';

  @override
  String get delete_summit_log_confirm =>
      'Vil du virkelig slette denne toppboken? Denne handlingen kan ikke angres.';

  @override
  String get delete_trail_confirm =>
      'Vil du virkelig slette denne stien? Denne handlingen kan ikke angres.';

  @override
  String get describe_your_trail => 'Beskriv stien din';

  @override
  String get description => 'Beskrivelse';

  @override
  String get difficult => 'Vanskelig';

  @override
  String get difficulty => 'Vanskelighetsgrad';

  @override
  String get directions => 'Veibeskrivelse';

  @override
  String get display => 'Visning';

  @override
  String get display_as => 'Vis som';

  @override
  String get distance => 'Distanse';

  @override
  String get documentation => 'Dokumentasjon';

  @override
  String get download => 'Last ned';

  @override
  String get draw_a_route => 'Tegn en rute';

  @override
  String get driving => 'Kjøring';

  @override
  String get duplicate => 'Dupliser';

  @override
  String get duration => 'Varighet';

  @override
  String get dutch => 'Nederlandsk';

  @override
  String get easy => 'Lett';

  @override
  String get edit => 'Rediger';

  @override
  String get edit_entry => 'Rediger oppføring';

  @override
  String get edit_list => 'Rediger liste';

  @override
  String get edit_route => 'Rediger rute';

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
  String get email_verified => 'E-post verifisert';

  @override
  String empty_activities(Object username) {
    return '$username har ingen aktivitet ennå';
  }

  @override
  String empty_bio(Object username) {
    return '$username har ikke lagt til en biografi ennå';
  }

  @override
  String get empty_feed => 'Strømmen din er tom';

  @override
  String get empty_feed_explanation =>
      'Aktiviteter fra deg eller folk du følger vil vises her';

  @override
  String empty_lists(Object username) {
    return '$username har ingen offentlige lister';
  }

  @override
  String get enable_auto_routing => 'Aktiver automatisk ruting';

  @override
  String get english => 'Engelsk';

  @override
  String get entry => 'Oppføring';

  @override
  String get error_copying_trail => 'Feil ved kopiering av sti';

  @override
  String get error_creating_user => 'Feil ved oppretting av bruker';

  @override
  String get error_deleting_token => 'Feil ved sletting av token';

  @override
  String get error_disabling_strava_integration =>
      'Feil ved deaktivering av Strava-integrasjon';

  @override
  String get error_during_login => 'Feil under innlogging';

  @override
  String get error_during_password_reset =>
      'Kunne ikke sende e-post for tilbakestilling av passord';

  @override
  String get error_exporting_trail => 'Feil ved eksport av sti';

  @override
  String get error_generating_token => 'Feil ved generering av token';

  @override
  String get error_liking_trail => 'Feil ved likerklikk på sti';

  @override
  String get error_logging_in_to_hammerhead =>
      'Feil ved innlogging på Hammerhead';

  @override
  String get error_logging_in_to_komoot => 'Feil ved innlogging til Komoot';

  @override
  String get error_posting_comment => 'Feil ved posting av kommentar';

  @override
  String get error_printing_map => 'Feil ved utskrift av kart';

  @override
  String get error_reading_file => 'Feil ved lesing av fil';

  @override
  String get error_saving_list => 'Feil ved lagring av liste';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Feil ved lagring av sti';

  @override
  String error_setting_up_integration(Object provider) {
    return 'Feil ved oppsett av $provider-integrasjon';
  }

  @override
  String get error_updating_hammerhead_integration =>
      'Feil ved oppdatering av Hammerhead-integrasjon';

  @override
  String get error_updating_komoot_integration =>
      'Feil ved oppdatering av Komoot-integrasjon';

  @override
  String get error_updating_password => 'Feil ved oppdatering av passord';

  @override
  String get error_updating_strava_integration =>
      'Feil ved oppdatering av Komoot-integrasjon';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Feil ved opplasting av sti til Hammerhead';

  @override
  String get est_duration => 'Est. varighet';

  @override
  String get everyone_with_the_link => 'Alle med lenken';

  @override
  String get expand_trail_list => 'Vis stiliste';

  @override
  String get expiration => 'Utløp';

  @override
  String get expires => 'Utløper';

  @override
  String get explore => 'Utforsk';

  @override
  String get explore_some_trails => 'Utforsk noen stier';

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
  String get export => 'Eksporter';

  @override
  String get export_all_trails => 'Eksporter alle stier';

  @override
  String get favourite_sport => 'Favorittsport';

  @override
  String get features => 'Funksjoner';

  @override
  String get ferry => 'Ferge';

  @override
  String get file_format => 'Filformat';

  @override
  String file_too_big(Object file, Object size) {
    return 'Filen $file er for stor (maks $size)';
  }

  @override
  String get filter_categories => 'Filtrer kategorier';

  @override
  String get filter_difficulty => 'Filtrer vanskelighetsgrad';

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
  String get fixed_speed => 'Fast hastighet';

  @override
  String get focus_map_on => 'Fokuser kart på';

  @override
  String get follow => 'Følg';

  @override
  String get follow_request_pending => 'Forespørsel venter';

  @override
  String get followers => 'Følgere';

  @override
  String get following => 'Følger';

  @override
  String get food => 'Mat';

  @override
  String get food_drinks => 'Mat og drikke';

  @override
  String get forgot_your_password => 'Glemt passordet?';

  @override
  String get french => 'Fransk';

  @override
  String get from_file => 'Fra fil';

  @override
  String get from_photos => 'Fra bilder';

  @override
  String get from_url => 'Fra URL';

  @override
  String get garage => 'Garasje';

  @override
  String get gas_station => 'Bensinstasjon';

  @override
  String get generate_new_token => 'Generer nytt token';

  @override
  String get german => 'Tysk';

  @override
  String get get_position_from_exif => 'Hent koordinater fra EXIF-data';

  @override
  String get get_started => 'Kom i gang';

  @override
  String get grid => 'Rutenett';

  @override
  String get grocery_store => 'Dagligvarebutikk';

  @override
  String get hammerhead_integration_after_date_hint =>
      'Hvis Hammerhead-kontoen din allerede er synkronisert med andre stidatabaser, som Komoot eller Strava, kan synkronisering av Hammerhead-data føre til duplikater. For å unngå dette kan du angi en startdato nedenfor, slik at bare aktiviteter registrert etter denne datoen vil bli synkronisert.';

  @override
  String get heading => 'Overskrift';

  @override
  String get height => 'Høyde';

  @override
  String get help => 'Hjelp';

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
  String get hiking => 'Vandring';

  @override
  String get home => 'Hjem';

  @override
  String get hotel => 'Hotell';

  @override
  String get hungarian => 'Ungarsk';

  @override
  String get hut => 'Hytte';

  @override
  String get hybrid => 'Hybrid';

  @override
  String get icon => 'Ikon';

  @override
  String get ignore_trails_before_date => 'Ignorer stier før denne datoen';

  @override
  String get imperial => 'Imperisk';

  @override
  String get import => 'Importer';

  @override
  String get import_hint =>
      'Velg eller dra GPX, FIT, KML eller TCX-filer hit...';

  @override
  String get include_description => 'Inkluder beskrivelse';

  @override
  String get include_waypoints => 'Inkluder veipunkter';

  @override
  String get integration_description_hammerhead =>
      'Synkroniserer Hammerhead-turene dine med Wanderer med jevne mellomrom.';

  @override
  String get integration_description_komoot =>
      'Synkroniserer dine Komoot-turer med Wanderer med jevne mellomrom.';

  @override
  String get integration_description_strava =>
      'Synkroniserer dine Strava-ruter og aktiviteter med Wanderer med jevne mellomrom.';

  @override
  String get integration_disabled => 'integrasjon deaktivert';

  @override
  String get integration_enabled => 'integrasjon aktivert';

  @override
  String get integration_privacy_hint_original =>
      'Importerte stier vil beholde samme synlighet som de har på den eksterne plattformen. For eksempel, hvis den opprinnelige stien var offentlig, vil den være offentlig i Wanderer, selv om stier er private som standard i henhold til personverninnstillingene dine.';

  @override
  String get integration_privacy_hint_user =>
      'Den opprinnelige stiens synlighet blir forkastet. I stedet blir de lokale personverninnstillingene for stier brukt på alle importerte stier.';

  @override
  String get integrations => 'Integrasjoner';

  @override
  String get invalid_date => 'Ugyldig dato';

  @override
  String get invalid_username => 'Ugyldig brukernavn';

  @override
  String get italian => 'Italiensk';

  @override
  String get joined => 'Ble medlem';

  @override
  String get keep_original => 'Behold original';

  @override
  String get keep_private => 'Hold privat';

  @override
  String get language => 'Språk';

  @override
  String get last_used => 'Sist brukt';

  @override
  String get latitude => 'Breddegrad';

  @override
  String layer(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Lag',
      one: 'Lag',
    );
    return '$_temp0';
  }

  @override
  String get license => 'Lisens';

  @override
  String get like_status => 'Liker-status';

  @override
  String get liked => 'Likte';

  @override
  String get likes => 'Likerklikk';

  @override
  String get limited => 'Begrenset';

  @override
  String get link_copied => 'Lenke kopiert!';

  @override
  String get linked_lists => 'Koblede lister';

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
  String get list_not_shared => 'Ikke delt med noen';

  @override
  String get list_public_warning =>
      'Alle stier i denne listen vil bli offentlige.';

  @override
  String get list_saved_successfully => 'Liste lagret';

  @override
  String get list_share_warning =>
      'Deling av en liste deler automatisk alle stier i den.';

  @override
  String get list_share_warning_update =>
      'Tillagte stier vil bli delt med alle som har tilgang til denne listen.';

  @override
  String get location => 'Plassering';

  @override
  String get locations => 'Locations';

  @override
  String get location_tracking_notification_title => 'Wanderer';

  @override
  String get location_tracking_notification_text => 'Recording your trail';

  @override
  String get login => 'Logg inn';

  @override
  String get login_details => 'Innloggingsdetaljer';

  @override
  String get logout => 'Logg ut';

  @override
  String get longitude => 'Lengdegrad';

  @override
  String get loop => 'Runde';

  @override
  String get make_one => 'Lag en!';

  @override
  String get make_thumbnail => 'Lag miniatyrbilde';

  @override
  String get map => 'Kart';

  @override
  String get map_style => 'Kartstil';

  @override
  String get max_hiking_difficulty => 'Maks. vanskelighetsgrad';

  @override
  String get metric => 'Metrisk';

  @override
  String get moderate => 'Moderat';

  @override
  String get more => 'Mer';

  @override
  String get more_route_settings => 'Flere ruteinnstillinger';

  @override
  String get mountain => 'Fjell';

  @override
  String get mountain_pass => 'Fjellovergang';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return 'Må være minst $n tegn lang';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return 'Må være maks $n tegn lang';
  }

  @override
  String get my_account => 'Min konto';

  @override
  String get my_trails => 'Mine stier';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return '$n dager siden';
  }

  @override
  String n_hours_ago(Object n) {
    return '$n timer siden';
  }

  @override
  String n_minutes_ago(Object n) {
    return '$n minutter siden';
  }

  @override
  String n_months_ago(Object n) {
    return '$n måneder siden';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n sekunder siden',
      zero: 'akkurat nå',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return '$n år siden';
  }

  @override
  String get name => 'Navn';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => 'Nær';

  @override
  String get never => 'Aldri';

  @override
  String get new_list => 'Ny liste';

  @override
  String get new_password => 'Nytt passord';

  @override
  String get new_password_error => 'Feil ved setting av nytt passord';

  @override
  String get new_password_success => 'Det nye passordet er satt';

  @override
  String get new_password_text => 'Sett et nytt passord';

  @override
  String get new_token_generated => 'Nytt API-token generert';

  @override
  String get new_trail => 'Ny sti';

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
  String get no_account => 'Har du ikke en konto?';

  @override
  String get no_api_tokens => 'Du har ingen API-tokens';

  @override
  String get no_comments_so_far => 'Ingen kommentarer ennå';

  @override
  String get no_data => 'Ingen data';

  @override
  String get no_description_for_now => 'Ingen beskrivelse ennå';

  @override
  String get no_gps_data_in_image => 'Ingen GPS-data i bildet';

  @override
  String get no_grid => 'Ingen rutenett';

  @override
  String get no_notifications => 'Ingen varsler';

  @override
  String get no_photos_here => 'Ingen bilder eller videoer her';

  @override
  String get no_preference => 'Ingen preferanse';

  @override
  String get no_results => 'Ingen resultater funnet';

  @override
  String get no_routes_added => 'Ingen ruter lagt til';

  @override
  String get no_waypoints_yet => 'Ingen veipunkter ennå';

  @override
  String get norwegian => 'Norsk';

  @override
  String get not_a_valid_email_address => 'Ikke en gyldig e-postadresse';

  @override
  String get not_a_valid_url => 'Ikke en gyldig URL';

  @override
  String get not_completed => 'Ikke fullført';

  @override
  String notification_comment_mention(Object user) {
    return '$user nevnte deg i en kommentar';
  }

  @override
  String notification_list_create(Object user) {
    return '$user opprettet en ny liste';
  }

  @override
  String notification_list_share(Object user) {
    return '$user delte en liste med deg';
  }

  @override
  String get notification_new_follower => 'Du har en ny følger';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user opprettet en toppbokelement på din sti \"$trail\"';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user nevnte deg i en toppbokelement';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user la igjen en kommentar på din sti \"$trail\"';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user opprettet en ny sti';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user likte din sti \"$trail\"';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user nevnte deg i en sti';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user delte en sti med deg';
  }

  @override
  String get notifications => 'Varsler';

  @override
  String object_share_error(Object object) {
    return 'Et $object må være offentlig for å bli delt på tvers av instanser.';
  }

  @override
  String get off => 'Av';

  @override
  String get only_me => 'Kun meg';

  @override
  String get open_in_new_tab => 'Åpne i ny fane';

  @override
  String get or => 'eller';

  @override
  String get orientation => 'Orientering';

  @override
  String get paper_size => 'Papirstørrelse';

  @override
  String get paragraph => 'Avsnitt';

  @override
  String get parking => 'Parkering';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Passord';

  @override
  String get password_confirm => 'Bekreft passord';

  @override
  String get password_reset_sent =>
      'En e-post for tilbakestilling av passord er sendt';

  @override
  String get password_reset_text =>
      'Vi sender en lenke for tilbakestilling til din e-postadresse.';

  @override
  String get password_updated => 'Passord oppdatert';

  @override
  String get passwords_must_match => 'Passordene må være like';

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
  String get pick_a_trail => 'Velg en sti';

  @override
  String get planned_a_trail => 'planla en sti';

  @override
  String get planned_tours => 'Planlagte turer';

  @override
  String get pois => 'Interessepunkter';

  @override
  String get polish => 'Polsk';

  @override
  String get portuguese => 'Portugisisk';

  @override
  String get print => 'Skriv ut';

  @override
  String get privacy => 'Personvern';

  @override
  String get private => 'Privat';

  @override
  String get profile => 'Profil';

  @override
  String get public => 'Offentlig';

  @override
  String get public_access => 'Offentlig tilgang';

  @override
  String get public_share_everyone =>
      'Alle på internett med lenken kan se denne stien';

  @override
  String get public_share_limited => 'Kun personer med tilgang kan åpne lenken';

  @override
  String get public_transport => 'Kollektivtransport';

  @override
  String get radius => 'Radius';

  @override
  String get railway_station => 'Jernbanestasjon';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get read_more => 'Les mer';

  @override
  String get ready_to_join => 'Klar til å bli med';

  @override
  String get recalculate_elevation_data => 'Beregn høydedata på nytt';

  @override
  String get recalculating_elevation_data_hint =>
      'Ny beregning av høydedata vil slette eksisterende høydedata, hvis noen, og erstatte det med data fra Valhalla.';

  @override
  String get register => 'Registrer';

  @override
  String get remote_users_cannot_edit => 'Eksterne brukere kan ikke redigere';

  @override
  String get removed_trail_from => 'Fjernet sti fra';

  @override
  String get removed_trails_from => 'Fjernet stier fra';

  @override
  String get required => 'Påkrevd';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Tilbakestill';

  @override
  String get reset_password => 'Tilbakestill passord';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => 'Gjenoppta opptaket?';

  @override
  String get reverse_direction => 'Snu retning';

  @override
  String get road => 'Vei';

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
  String get route_point => 'Rutepunkt';

  @override
  String get russian => 'Russisk';

  @override
  String get save => 'Lagre';

  @override
  String get save_list => 'Lagre liste';

  @override
  String get save_track => 'Save track';

  @override
  String get save_trail => 'Lagre sti';

  @override
  String get save_your_trail_first => 'Lagre stien din først';

  @override
  String get search => 'Search';

  @override
  String get search_cities => 'Søk etter byer';

  @override
  String get search_for_trails_places => 'Søk etter stier, lister, steder';

  @override
  String get search_list => 'Søk i liste';

  @override
  String get search_places => 'Søk etter steder';

  @override
  String get search_trails => 'Søk etter stier';

  @override
  String get select_date => 'Select date';

  @override
  String get select_list => 'Velg liste';

  @override
  String get selected => 'valgt';

  @override
  String get send_to => 'Send til...';

  @override
  String get set_private => 'Sett som privat';

  @override
  String get set_public => 'Sett som offentlig';

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
  String get settings_saved => 'Innstillinger lagret';

  @override
  String get share => 'Del';

  @override
  String get share_profile => 'Del profil';

  @override
  String get share_this_list => 'Del denne listen';

  @override
  String get share_this_trail => 'Del denne stien';

  @override
  String get shared => 'Delt';

  @override
  String get shared_by => 'Delt av';

  @override
  String get shared_with => 'Delt med';

  @override
  String get shelter => 'Gapahuk';

  @override
  String get shortest => 'kortest';

  @override
  String get show_in_overview => 'Vis i oversikt';

  @override
  String get show_less => 'Vis mindre';

  @override
  String get show_on_map => 'Vis på kart';

  @override
  String get shower => 'Dusj';

  @override
  String get skiing => 'Skisport';

  @override
  String get slogan => 'Lagre dine eventyr!';

  @override
  String get slope => 'Helning';

  @override
  String get someone => 'Noen';

  @override
  String get sort => 'Sorter';

  @override
  String get spanish => 'Spansk';

  @override
  String get speed => 'Hastighet';

  @override
  String get start => 'Start';

  @override
  String get statistics => 'Statistikk';

  @override
  String get stop_drawing => 'Stopp tegning';

  @override
  String get stop_editing => 'Stopp redigering';

  @override
  String get strava_integration_after_date_hint =>
      'Hvis kontoen din har en stor mengde aktiviteter kan du støte på Stravas API-hastighetsgrense som hindrer deg i å synkronisere alle aktiviteter samtidig. For å redusere dette problemet kan du sette en \"Etter\" dato nedenfor slik at bare aktiviteter som ble registrert etter denne datoen blir synkronisert.';

  @override
  String get subcategories => 'Underkategorier';

  @override
  String get subway_stop => 'T-baneinngang';

  @override
  String get summit => 'Topp';

  @override
  String get summit_book => 'Toppbok';

  @override
  String get table => 'Tabell';

  @override
  String get tags => 'Tagger';

  @override
  String get text => 'Tekst';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get tilesets => 'Tilpassede flissett';

  @override
  String get time => 'Time';

  @override
  String get time_in_motion => 'Time in Motion';

  @override
  String get toilets => 'Toaletter';

  @override
  String get top_speed => 'Toppfart';

  @override
  String get tourism => 'Turisme';

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
  String get trail_copied_successfully => 'sti kopiert';

  @override
  String get trail_has_no_gpx => 'Denne stien har ingen GPX-data.';

  @override
  String get trail_not_in_list => 'Stien er ikke i noen liste';

  @override
  String get trail_not_shared => 'Ikke delt med noen';

  @override
  String get trail_saved_successfully => 'Sti lagret';

  @override
  String get some_waypoints_failed_to_save =>
      'Trail saved, but some waypoints failed to save';

  @override
  String get trails_for_you => 'Stier for deg';

  @override
  String get tram_stop => 'Trikkestopp';

  @override
  String get unchanged => 'uendret';

  @override
  String get units => 'Enheter';

  @override
  String get unlink => 'Frakoble';

  @override
  String get upload_file => 'Last opp fil';

  @override
  String get upload_gpx => 'Last opp GPX';

  @override
  String get upload_new_file => 'Last opp ny fil';

  @override
  String get uploaded => 'lastet opp';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Stien ble lastet opp til Hammerhead';

  @override
  String get use_hills => 'Bruk bakker';

  @override
  String get use_roads => 'Bruk veier';

  @override
  String get users => 'Users';

  @override
  String get username => 'Brukernavn';

  @override
  String get username_not_unique =>
      'Dette brukernavnet er allerede tatt. Vennligst prøv et annet.';

  @override
  String get view => 'Vis';

  @override
  String get viewpoint => 'Utsiktspunkt';

  @override
  String get visibilty => 'Synlighet';

  @override
  String get visibilty_status => 'Synlighetsstatus';

  @override
  String get walking_speed => 'Gåhastighet';

  @override
  String get water => 'Vann';

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
  String get width => 'Bredde';

  @override
  String get wrong_username_or_password => 'Feil brukernavn eller passord';

  @override
  String get you_can => 'Du kan';

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
