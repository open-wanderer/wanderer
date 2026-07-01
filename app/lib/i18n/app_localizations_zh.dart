// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get biking => 'Biking';

  @override
  String get canoeing => 'Canoeing';

  @override
  String get walking => 'Walking';

  @override
  String get about => '关于';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm => '您现在要删除当前账户，所有的路线都会删除无法恢复，确认继续操作吗？';

  @override
  String get account_privacy => '账户隐私';

  @override
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '活动',
      one: '活动',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => '添加个人信息';

  @override
  String get add_entry => '添加日程';

  @override
  String get add_to_list => '添加到列表';

  @override
  String get add_waypoint => '添加坐标';

  @override
  String get added_trail_to => '添加路线到';

  @override
  String get added_trails_to => '添加路线到';

  @override
  String get after => '之后';

  @override
  String get all => 'All';

  @override
  String get all_activities => '所有活动';

  @override
  String get allow_auto_geolocate =>
      'Begin drawing a new trail from your current location';

  @override
  String get alphabetical => '字母';

  @override
  String get already_account => '已注册账户？';

  @override
  String get altitude => '海拔';

  @override
  String get amenity => '友好性';

  @override
  String get api_documentation => 'API 文档';

  @override
  String get api_tokens => 'API Tokens';

  @override
  String get api_tokens_hint =>
      'API Tokens can be used to grant 3rd party applications access to your wanderer account.';

  @override
  String get apply_user_settings => 'Apply user settings';

  @override
  String get attraction => '景点';

  @override
  String get author => '作者';

  @override
  String get avatar => '头像';

  @override
  String get average_speed => '平均速度';

  @override
  String get avoid_bad_surfaces => '避免损坏的路面';

  @override
  String get back => '返回';

  @override
  String get back_to_login => '返回登录';

  @override
  String get bakery => '面包店';

  @override
  String get barrier => '障碍';

  @override
  String get basic_info => '基本信息';

  @override
  String get basque => 'Basque';

  @override
  String get before => '之前';

  @override
  String get behavior => 'Behavior';

  @override
  String get bicycle_parking => '自行车停车场';

  @override
  String get bicycle_rental => '自行车租车';

  @override
  String get bicycle_shop => '自行车店';

  @override
  String get bike_type => '自行车类型';

  @override
  String get bus_stop => '公交车站';

  @override
  String get by => '由';

  @override
  String get campsite => '营地';

  @override
  String get can => '可以';

  @override
  String get cancel => '取消';

  @override
  String get car => '汽车';

  @override
  String get car_motorcycle => '汽车/摩托车';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '卡片',
      one: '卡片',
    );
    return '$_temp0';
  }

  @override
  String get categories => '分类';

  @override
  String get category => '分类';

  @override
  String get change => '更换';

  @override
  String get change_email => '更改邮箱';

  @override
  String get change_password => '更改密码';

  @override
  String get changelog => '变更日志';

  @override
  String get chinese => '中文（简体）';

  @override
  String get clear_all => '清除全部';

  @override
  String get climbing => '攀爬';

  @override
  String get close => '关闭';

  @override
  String get collapse_trail_list => 'Collapse trail list';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '评论',
      one: '评论',
    );
    return '$_temp0';
  }

  @override
  String get completed => '已完成';

  @override
  String get completed_a_trail => '完成轨迹';

  @override
  String get completed_tours => '已完成的旅行';

  @override
  String get completion_status => '完成状态';

  @override
  String get confirm => '确认';

  @override
  String get confirm_deletion => '确认删除';

  @override
  String get confirm_publish => '确认发布';

  @override
  String get confirm_share => '确认分享';

  @override
  String get connect => '连接';

  @override
  String get contribute => '贡献';

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
  String get copy_link => '复制链接';

  @override
  String get create_new_list => '创建新列表';

  @override
  String get create_waypoint => '创建路径点';

  @override
  String get creation_date => '创建日期';

  @override
  String get crop => '裁剪';

  @override
  String get cross => '交叉点';

  @override
  String get current_password => '当前密码';

  @override
  String get cycling => '骑行';

  @override
  String get cycling_speed => '骑车速度';

  @override
  String get czech => 'Czech';

  @override
  String get danger_zone => '危险区域';

  @override
  String get date => '日期';

  @override
  String get default_category => '默认分类';

  @override
  String get default_location => '默认地点';

  @override
  String get degrees => '度';

  @override
  String get delete => '删除';

  @override
  String get open => 'Open';

  @override
  String get delete_account => '注销账户';

  @override
  String get delete_list_confirm => '您确认要删除当前列表？列表中所有行程仍然继续保留不会被删除。';

  @override
  String get delete_summit_log_confirm => '您确认想要删除当前行程记录？此操作不可撤回。';

  @override
  String get delete_trail_confirm => '您确认想要删除当前行程？此操作不可撤回。';

  @override
  String get describe_your_trail => '行程描述';

  @override
  String get description => '描述';

  @override
  String get difficult => '困难';

  @override
  String get difficulty => '难度';

  @override
  String get directions => '导航';

  @override
  String get display => '显示';

  @override
  String get display_as => '显示方式';

  @override
  String get distance => '距离';

  @override
  String get documentation => '文档';

  @override
  String get download => '下载';

  @override
  String get draw_a_route => '绘制路线';

  @override
  String get driving => '驾驶';

  @override
  String get duplicate => '复制';

  @override
  String get duration => '持续时间';

  @override
  String get dutch => '荷兰语';

  @override
  String get easy => '简单';

  @override
  String get edit => '编辑';

  @override
  String get edit_entry => '编辑日程';

  @override
  String get edit_list => '编辑列表';

  @override
  String get edit_route => '编辑路线';

  @override
  String get edit_waypoint => '编辑坐标';

  @override
  String get edited => '已编辑';

  @override
  String get elevation_gain => '上升海拔';

  @override
  String get elevation_loss => '下降海拔';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => '电子邮箱';

  @override
  String get email_not_unique => 'This email address is already in use.';

  @override
  String get email_updated => '邮箱已更改';

  @override
  String get email_verified => '邮箱已经验证';

  @override
  String empty_activities(Object username) {
    return '$username还没有活动';
  }

  @override
  String empty_bio(Object username) {
    return '$username还没添加简介';
  }

  @override
  String get empty_feed => '您的订阅是空的';

  @override
  String get empty_feed_explanation => '您或您关注的人的活动将出现在这里';

  @override
  String empty_lists(Object username) {
    return '$username未有公开的列表';
  }

  @override
  String get enable_auto_routing => '启用自动路由';

  @override
  String get english => '英语';

  @override
  String get entry => '日程';

  @override
  String get error_copying_trail => 'Error copying trail';

  @override
  String get error_creating_user => '创建用户错误';

  @override
  String get error_deleting_token => 'Error deleting token';

  @override
  String get error_disabling_strava_integration => '禁用strava集成时出错';

  @override
  String get error_during_login => '登录错误';

  @override
  String get error_during_password_reset => '无法发送密码重置邮件';

  @override
  String get error_exporting_trail => '导出路线失败';

  @override
  String get error_generating_token => 'Error generating token';

  @override
  String get error_liking_trail => '赞轨迹时出错';

  @override
  String get error_logging_in_to_hammerhead => 'Error logging in to Hammerhead';

  @override
  String get error_logging_in_to_komoot => '登录到 komoot 时出错';

  @override
  String get error_posting_comment => '发布评论时出错';

  @override
  String get error_printing_map => '打印地图失败';

  @override
  String get error_reading_file => '读取文件错误';

  @override
  String get error_saving_list => '保存列表失败';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => '保存路线失败';

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
  String get error_updating_password => '更新密码失败';

  @override
  String get error_updating_strava_integration => '更新 komoot 集成出错';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Error uploading trail to Hammerhead';

  @override
  String get est_duration => '预计时长';

  @override
  String get everyone_with_the_link => 'Everyone with the link';

  @override
  String get expand_trail_list => 'Expand trail list';

  @override
  String get expiration => 'Expiration';

  @override
  String get expires => 'Expires';

  @override
  String get explore => '探索';

  @override
  String get explore_some_trails => '探索行程';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get export => '导出';

  @override
  String get export_all_trails => '导出所有路线';

  @override
  String get favourite_sport => '喜欢的运动';

  @override
  String get features => '特性';

  @override
  String get ferry => '轮渡';

  @override
  String get file_format => '文件格式';

  @override
  String file_too_big(Object file, Object size) {
    return '$file文件太大了（最大$size）';
  }

  @override
  String get filter_categories => '筛选分类';

  @override
  String get filter_difficulty => '筛选难度';

  @override
  String get filter_tags => '过滤标签';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => '完成';

  @override
  String get fixed_speed => '固定速度';

  @override
  String get focus_map_on => '地图聚焦于';

  @override
  String get follow => '关注';

  @override
  String get follow_request_pending => '请求待处理';

  @override
  String get followers => '关注者';

  @override
  String get following => '已关注';

  @override
  String get food => '食品';

  @override
  String get food_drinks => '食物和饮料';

  @override
  String get forgot_your_password => '忘记密码？';

  @override
  String get french => '法语';

  @override
  String get from_file => '来自文件';

  @override
  String get from_photos => '来自照片';

  @override
  String get from_url => '从 URL';

  @override
  String get garage => '车库';

  @override
  String get gas_station => '加油站';

  @override
  String get generate_new_token => 'Generate new token';

  @override
  String get german => '德语';

  @override
  String get get_position_from_exif => '从EXIF数据获取坐标';

  @override
  String get get_started => 'Get started';

  @override
  String get grid => '网格';

  @override
  String get grocery_store => '杂货店';

  @override
  String get hammerhead_integration_after_date_hint =>
      'If your hammerhead account is already synced with other trail databases, such as komoot or Strava, start syncing your Hammerhead data may result in duplicates. To avoid this, you can set an start date below, meaning only activities recorded after this date will be synced.';

  @override
  String get heading => '标题';

  @override
  String get height => '高度';

  @override
  String get help => '帮助';

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
  String get hiking => '徒步';

  @override
  String get home => '主页';

  @override
  String get hotel => '酒店';

  @override
  String get hungarian => '匈牙利语';

  @override
  String get hut => '小屋';

  @override
  String get hybrid => '混合';

  @override
  String get icon => '图标';

  @override
  String get ignore_trails_before_date => 'Ignore trails before this date';

  @override
  String get imperial => '英制';

  @override
  String get import => '导入';

  @override
  String get import_hint => '在此选择或拖拽GPX、FIT、KML或TCX文件...';

  @override
  String get include_description => '包含描述';

  @override
  String get include_waypoints => '包括途径点';

  @override
  String get integration_description_hammerhead =>
      'Syncs your Hammerhead tours with wanderer in regular intervals.';

  @override
  String get integration_description_komoot => '定期与komoot同步您的wanderer。';

  @override
  String get integration_description_strava => '定期与strava同步您的wanderer路线和活动。';

  @override
  String get integration_disabled => '整合已停用';

  @override
  String get integration_enabled => '整合已启用';

  @override
  String get integration_privacy_hint_original =>
      'Imported trails will maintain the same visibility they have on the external platform. For example, if the original trail was public, it will be public in wanderer, even if trails are private by default according to your privacy settings.';

  @override
  String get integration_privacy_hint_user =>
      'The original trail\'s visibility is discarded. Instead, the local privacy settings for trails are applied to all imported trails.';

  @override
  String get integrations => '整合';

  @override
  String get invalid_date => '无效日期';

  @override
  String get invalid_username => '无效用户名';

  @override
  String get italian => '意大利语';

  @override
  String get joined => '已加入';

  @override
  String get keep_original => 'Keep original';

  @override
  String get keep_private => 'Keep private';

  @override
  String get language => '语言';

  @override
  String get last_used => 'Last used';

  @override
  String get latitude => '纬度';

  @override
  String layer(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '层',
      one: '层',
    );
    return '$_temp0';
  }

  @override
  String get license => '开源协议';

  @override
  String get like_status => '喜欢状态';

  @override
  String get liked => '赞';

  @override
  String get likes => '赞';

  @override
  String get limited => 'Limited';

  @override
  String get link_copied => '链接已复制';

  @override
  String get linked_lists => 'Linked lists';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '列表',
      one: '列表',
    );
    return '$_temp0';
  }

  @override
  String get list_not_shared => '未与任何人分享';

  @override
  String get list_public_warning => '此列表中的所有轨迹都将被公开。';

  @override
  String get list_saved_successfully => '列表保存成功';

  @override
  String get list_share_warning => '分享列表会自动分享其中包含的所有路线';

  @override
  String get list_share_warning_update => '添加的路线将与所有可访问此列表的人分享';

  @override
  String get location => '地点';

  @override
  String get locations => 'Locations';

  @override
  String get login => '登录';

  @override
  String get login_details => '登录详情';

  @override
  String get logout => '登出';

  @override
  String get longitude => '经度';

  @override
  String get loop => '循环';

  @override
  String get make_one => '立刻注册！';

  @override
  String get make_thumbnail => '生成缩略图';

  @override
  String get map => '地图';

  @override
  String get map_style => '地图样式';

  @override
  String get max_hiking_difficulty => '最大徒步难度';

  @override
  String get metric => '公制';

  @override
  String get moderate => '中等';

  @override
  String get more => '更多';

  @override
  String get more_route_settings => '更多路由设置';

  @override
  String get mountain => '山地';

  @override
  String get mountain_pass => '山地通过';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return '长度至少 $n 字符';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return '长度至少 $n 字符';
  }

  @override
  String get my_account => '我的账户';

  @override
  String get my_trails => '我的轨迹';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return '$n 天前';
  }

  @override
  String n_hours_ago(Object n) {
    return '$n 小时前';
  }

  @override
  String n_minutes_ago(Object n) {
    return '$n 分钟前';
  }

  @override
  String n_months_ago(Object n) {
    return '$n 个月前';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n 几秒前',
      zero: '刚才',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return '$n 年前';
  }

  @override
  String get name => '名称';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => '附近';

  @override
  String get never => 'Never';

  @override
  String get new_list => '新列表';

  @override
  String get new_password => '新密码';

  @override
  String get new_password_error => '设置新密码时出错';

  @override
  String get new_password_success => '新密码已设置';

  @override
  String get new_password_text => '设置新密码';

  @override
  String get new_token_generated => 'New API Token generated';

  @override
  String get new_trail => '创建新路线';

  @override
  String get no_account => '还未注册？';

  @override
  String get no_api_tokens => 'You have no API Tokens';

  @override
  String get no_comments_so_far => '到目前为止没有评论';

  @override
  String get no_data => '无数据';

  @override
  String get no_description_for_now => '暂无描述';

  @override
  String get no_gps_data_in_image => '图像中没有GPS数据';

  @override
  String get no_grid => '无网格';

  @override
  String get no_notifications => '没有通知';

  @override
  String get no_photos_here => 'No photos here';

  @override
  String get no_preference => '尚未规划';

  @override
  String get no_results => '没有找到结果';

  @override
  String get no_routes_added => '未添加路由';

  @override
  String get no_waypoints_yet => '尚无路点';

  @override
  String get norwegian => 'Norwegian';

  @override
  String get not_a_valid_email_address => '无效电子邮箱地址';

  @override
  String get not_a_valid_url => '网址无效';

  @override
  String get not_completed => '未完成';

  @override
  String notification_comment_mention(Object user) {
    return '$user 在评论中提及了您';
  }

  @override
  String notification_list_create(Object user) {
    return '$user创建了一条新列表';
  }

  @override
  String notification_list_share(Object user) {
    return '$user与您分享了一个列表';
  }

  @override
  String get notification_new_follower => '你有一位新粉丝';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user在您的轨迹$trail上创建了一个summit日志\"';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user mentioned you in a summit log';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user 在您的轨迹上留了评论 \"$trail\"';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user创建了一条新的轨迹';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user 赞了您的轨迹 \"$trail\"';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user 在一条轨迹中提到了您';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user 与您分享了一条轨迹';
  }

  @override
  String get notifications => '通知';

  @override
  String object_share_error(Object object) {
    return '一个$object必须公开才能在不同实例间共享。';
  }

  @override
  String get off => '关闭';

  @override
  String get only_me => '仅自己';

  @override
  String get open_in_new_tab => '在新标签页中打开';

  @override
  String get or => '或';

  @override
  String get orientation => '方向';

  @override
  String get paper_size => '纸张大小';

  @override
  String get paragraph => '段落';

  @override
  String get parking => '停车场';

  @override
  String get pause => 'Pause';

  @override
  String get password => '密码';

  @override
  String get password_confirm => '确认密码';

  @override
  String get password_reset_sent => '密码重置邮件已发送';

  @override
  String get password_reset_text => '我们将发送一个重置链接到您的电子邮件地址。';

  @override
  String get password_updated => '密码已更新';

  @override
  String get passwords_must_match => '两次输入的密码必须一致';

  @override
  String get photos => '图片';

  @override
  String get pick_a_trail => '选择一个路线';

  @override
  String get planned_a_trail => '计划了一条轨迹';

  @override
  String get planned_tours => '计划的旅行';

  @override
  String get pois => '兴趣点';

  @override
  String get polish => '波兰语';

  @override
  String get portuguese => '葡萄牙语';

  @override
  String get print => '打印';

  @override
  String get privacy => '隐私';

  @override
  String get private => '非公开的';

  @override
  String get profile => '个人资料';

  @override
  String get public => '公开';

  @override
  String get public_access => 'Public access';

  @override
  String get public_share_everyone =>
      'Everyone on the internet with the link can see this trail';

  @override
  String get public_share_limited =>
      'Only people with access can open the link';

  @override
  String get public_transport => '公共交通';

  @override
  String get radius => '半径';

  @override
  String get railway_station => '火车站';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get read_more => '阅读更多';

  @override
  String get ready_to_join => 'Ready to join';

  @override
  String get recalculate_elevation_data => '重新计算海拔数据';

  @override
  String get recalculating_elevation_data_hint =>
      '重新计算海拔数据将会抹去现有的海拔数据，如果有的话，用Valhalla的数据取而代之。';

  @override
  String get register => '注册';

  @override
  String get remote_users_cannot_edit => '远程用户无法编辑';

  @override
  String get removed_trail_from => '路线已删除自';

  @override
  String get removed_trails_from => '路线已删除自';

  @override
  String get required => '必填';

  @override
  String get reset => '重置';

  @override
  String get reset_password => '重置密码';

  @override
  String get resume => 'Resume';

  @override
  String get reverse_direction => '反向';

  @override
  String get road => '道路';

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
  String get route_point => '路线点';

  @override
  String get russian => 'Russian';

  @override
  String get save => '保存';

  @override
  String get save_list => '保存列表';

  @override
  String get save_trail => '保存路线';

  @override
  String get save_your_trail_first => '先保存你的路线';

  @override
  String get search => 'Search';

  @override
  String get search_cities => '搜索城市';

  @override
  String get search_for_trails_places => '搜索路线、地点';

  @override
  String get search_list => 'Search list';

  @override
  String get search_places => '搜索地点';

  @override
  String get search_trails => '搜索路线';

  @override
  String get select_date => 'Select date';

  @override
  String get select_list => '选择列表';

  @override
  String get selected => '已选';

  @override
  String get send_to => 'Send to...';

  @override
  String get set_private => 'Set private';

  @override
  String get set_public => 'Set public';

  @override
  String get settings => '设置';

  @override
  String get settings_notification_comment_mention => '有人在评论中提到您';

  @override
  String get settings_notification_list_share => '有人与您分享了一个列表';

  @override
  String get settings_notification_new_follower => '您有一位新关注者';

  @override
  String get settings_notification_summit_log_create => '有人在您的轨迹上创建了一个summit日志';

  @override
  String get settings_notification_summit_log_mention => '有人在summit日志中提到了你';

  @override
  String get settings_notification_trail_comment => '有人在你的轨迹上留下了一条评论';

  @override
  String get settings_notification_trail_like => 'Somone liked your trail';

  @override
  String get settings_notification_trail_mention => '有人在轨迹中提到你';

  @override
  String get settings_notification_trail_share => '有人与您分享了一条轨迹';

  @override
  String get settings_privacy_account_private =>
      '只有您可以看到您的个人资料。您将不会出现在搜索结果中。 其他用户不能关注您或与您分享轨迹。您仍然可以发布轨迹或列表。';

  @override
  String get settings_privacy_account_public =>
      '每个人都可以看到您的个人资料。您将出现在搜索结果中。其他用户可以关注您并与您分享轨迹。';

  @override
  String get settings_privacy_lists_private =>
      '您的列表默认是私密的。只有您能够看到它们。 您可以在任何时候修改此设置。';

  @override
  String get settings_privacy_lists_public =>
      '您的列表默认是公开的。每个人都可以看到它们。您可以在任何时候更改此设置为个人列表使用。';

  @override
  String get settings_privacy_trails_private =>
      '您的轨迹默认是私密的。只有您能够看到它们。 您可以在任何时候修改此设置。';

  @override
  String get settings_privacy_trails_public =>
      '您的轨迹默认是公开的。每个人都可以看到它们。您可以在任何时候更改此设置为个人轨迹。';

  @override
  String get settings_saved => '设置已保存';

  @override
  String get share => '分享';

  @override
  String get share_profile => '分享个人资料';

  @override
  String get share_this_list => '分享此列表';

  @override
  String get share_this_trail => '分享此路线';

  @override
  String get shared => '共享';

  @override
  String get shared_by => '分享自';

  @override
  String get shared_with => '分享给';

  @override
  String get shelter => '庇护所';

  @override
  String get shortest => '最短的';

  @override
  String get show_in_overview => '详情中展示';

  @override
  String get show_less => '显示更少';

  @override
  String get show_on_map => '地图中展示';

  @override
  String get shower => '淋浴';

  @override
  String get skiing => '滑雪';

  @override
  String get slogan => '保存你的冒险！';

  @override
  String get slope => '坡度';

  @override
  String get someone => '某人';

  @override
  String get sort => '排序';

  @override
  String get spanish => '西班牙语';

  @override
  String get speed => '速度';

  @override
  String get start => '开始';

  @override
  String get statistics => '统计';

  @override
  String get stop_drawing => '停止绘制';

  @override
  String get stop_editing => '停止编辑';

  @override
  String get strava_integration_after_date_hint =>
      'If your account has a large amount of acitivities you may run into Strava\'s API rate limit preventing you from syncing all activities at once. To mitigate this issue you can set an \"After\" date below so that only activities that were recorded after this date are synced.';

  @override
  String get subcategories => '子分类';

  @override
  String get subway_stop => '地铁入口';

  @override
  String get summit => '山峰';

  @override
  String get summit_book => '详细日程';

  @override
  String get table => '表格';

  @override
  String get tags => '标签';

  @override
  String get text => '文本';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get tilesets => '自定义地图图层';

  @override
  String get time => 'Time';

  @override
  String get toilets => '厕所';

  @override
  String get top_speed => '最高速度';

  @override
  String get tourism => '旅游';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '路线',
      one: '路线',
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
  String get trail_not_shared => '未与任何人分享';

  @override
  String get trail_saved_successfully => '路线保存成功';

  @override
  String get trails_for_you => '推荐路线';

  @override
  String get tram_stop => '车站';

  @override
  String get unchanged => '未修改';

  @override
  String get units => '单位';

  @override
  String get unlink => '取消连接';

  @override
  String get upload_file => '上传文件';

  @override
  String get upload_gpx => '上传 GPX';

  @override
  String get upload_new_file => '上传新文件';

  @override
  String get uploaded => '已上传';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Successfully uploaded trail to Hammerhead';

  @override
  String get use_hills => '使用山地';

  @override
  String get use_roads => '使用道路';

  @override
  String get users => 'Users';

  @override
  String get username => '用户名';

  @override
  String get username_not_unique =>
      'This username is already taken. Please try another.';

  @override
  String get view => '查看';

  @override
  String get viewpoint => '视角';

  @override
  String get visibilty => 'Visibility';

  @override
  String get visibilty_status => '可见状态';

  @override
  String get walking_speed => '步行速度';

  @override
  String get water => '水';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '路点',
      one: '路点',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get width => '宽度';

  @override
  String get wrong_username_or_password => '用户名或密码无效';

  @override
  String get you_can => '你可以';

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
}
